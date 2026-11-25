import { tokenStorage } from '../auth/tokenStorage';

export interface ApiErrorEnvelope {
  error?: {
    code?: string;
    message?: string;
  };
}

export class ApiError extends Error {
  readonly status: number;
  readonly code: string;
  readonly retryAfterSeconds?: number;

  constructor(
    status: number,
    code: string,
    message: string,
    retryAfterSeconds?: number,
  ) {
    super(message);
    this.name = 'ApiError';
    this.status = status;
    this.code = code;
    this.retryAfterSeconds = retryAfterSeconds;
  }
}

export function isApiError(error: unknown): error is ApiError {
  return error instanceof ApiError;
}

export interface ApiRequestOptions {
  method?: 'GET' | 'POST' | 'PUT' | 'PATCH' | 'DELETE';
  body?: unknown;
  signal?: AbortSignal;
  headers?: HeadersInit;
  skipAuthHandling?: boolean;
}

export interface ApiClientConfig {
  baseUrl: string;
  getToken: () => string | null;
  onUnauthorized: () => void;
  fetchImpl?: typeof fetch;
}

function parseRetryAfter(headerValue: string | null): number | undefined {
  if (!headerValue) return undefined;
  const seconds = Number.parseInt(headerValue, 10);
  return Number.isFinite(seconds) && seconds >= 0 ? seconds : undefined;
}

function joinUrl(baseUrl: string, path: string): string {
  const normalizedBase = baseUrl.replace(/\/+$/, '');
  const normalizedPath = path.startsWith('/') ? path : `/${path}`;
  return `${normalizedBase}${normalizedPath}`;
}

async function parseErrorResponse(response: Response): Promise<ApiError> {
  const retryAfterSeconds = parseRetryAfter(response.headers.get('Retry-After'));
  const contentType = response.headers.get('Content-Type') ?? '';

  if (contentType.includes('application/json')) {
    try {
      const payload = (await response.json()) as ApiErrorEnvelope;
      const code = payload.error?.code ?? 'HTTP_ERROR';
      const message =
        payload.error?.message ?? (response.statusText || `HTTP ${response.status}`);
      return new ApiError(response.status, code, message, retryAfterSeconds);
    } catch {
      // Fall through to generic HTTP error when JSON is malformed.
    }
  }

  return new ApiError(
    response.status,
    'HTTP_ERROR',
    response.statusText || `HTTP ${response.status}`,
    retryAfterSeconds,
  );
}

export function createApiClient(config: ApiClientConfig) {
  const fetchFn = config.fetchImpl ?? ((...args: Parameters<typeof fetch>) => fetch(...args));

  async function request<T>(path: string, options: ApiRequestOptions = {}): Promise<T> {
    const method = options.method ?? 'GET';
    const headers = new Headers(options.headers);
    headers.set('Accept', 'application/json');

    const token = config.getToken();
    const sentToken = Boolean(token);
    if (token) {
      headers.set('Authorization', `Bearer ${token}`);
    }

    let serializedBody: string | undefined;
    if (options.body !== undefined) {
      headers.set('Content-Type', 'application/json');
      serializedBody = JSON.stringify(options.body);
    }

    const response = await fetchFn(joinUrl(config.baseUrl, path), {
      method,
      headers,
      body: serializedBody,
      signal: options.signal,
      credentials: 'omit',
    });

    if (!response.ok) {
      const apiError = await parseErrorResponse(response);
      if (response.status === 401 && sentToken && !options.skipAuthHandling) {
        config.onUnauthorized();
      }
      throw apiError;
    }

    if (response.status === 204) {
      return undefined as T;
    }

    const text = await response.text();
    if (!text) {
      return undefined as T;
    }

    return JSON.parse(text) as T;
  }

  return {
    request,
    get<T>(path: string, options?: Omit<ApiRequestOptions, 'method' | 'body'>) {
      return request<T>(path, { ...options, method: 'GET' });
    },
    post<T>(
      path: string,
      body?: unknown,
      options?: Omit<ApiRequestOptions, 'method' | 'body'>,
    ) {
      return request<T>(path, { ...options, method: 'POST', body });
    },
    put<T>(
      path: string,
      body?: unknown,
      options?: Omit<ApiRequestOptions, 'method' | 'body'>,
    ) {
      return request<T>(path, { ...options, method: 'PUT', body });
    },
    delete<T = void>(path: string, options?: Omit<ApiRequestOptions, 'method' | 'body'>) {
      return request<T>(path, { ...options, method: 'DELETE' });
    },
  };
}

const DEFAULT_API_URL = 'http://localhost:8080/api';

let unauthorizedCallback: (() => void) | null = null;

/**
 * Registers a callback invoked after the stored token is cleared on HTTP 401.
 * Allows the auth feature (T8) to redirect to /login without coupling client.ts to the router.
 */
export function setUnauthorizedHandler(handler: (() => void) | null): void {
  unauthorizedCallback = handler;
}

export const apiClient = createApiClient({
  baseUrl: import.meta.env.VITE_API_URL || DEFAULT_API_URL,
  getToken: () => tokenStorage.get(),
  onUnauthorized: () => {
    tokenStorage.clear();
    unauthorizedCallback?.();
  },
});
