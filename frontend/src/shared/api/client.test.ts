import { describe, expect, it, vi } from 'vitest';
import { ApiError, createApiClient } from './client';

function jsonResponse(body: unknown, init?: ResponseInit): Response {
  return new Response(JSON.stringify(body), {
    status: init?.status ?? 200,
    headers: {
      'Content-Type': 'application/json',
      ...(init?.headers ?? {}),
    },
  });
}

describe('createApiClient', () => {
  it('normalizes base URL and path and returns parsed JSON', async () => {
    const fetchMock = vi.fn<typeof fetch>().mockResolvedValue(
      jsonResponse({ status: 'ok' }),
    );
    const client = createApiClient({
      baseUrl: 'http://localhost:8080/api/',
      getToken: () => null,
      onUnauthorized: vi.fn(),
      fetchImpl: fetchMock,
    });

    const data = await client.get<{ status: string }>('/healthz');

    expect(data).toEqual({ status: 'ok' });
    expect(fetchMock).toHaveBeenCalledTimes(1);
    const [url, init] = fetchMock.mock.calls[0];
    expect(url).toBe('http://localhost:8080/api/healthz');
    expect(init?.method).toBe('GET');
    expect(init?.credentials).toBe('omit');
    const headers = new Headers(init?.headers);
    expect(headers.get('Accept')).toBe('application/json');
    expect(headers.get('Authorization')).toBeNull();
    expect(headers.get('Content-Type')).toBeNull();
  });

  it('attaches Bearer token when present and Content-Type when sending JSON body', async () => {
    const fetchMock = vi.fn<typeof fetch>().mockResolvedValue(
      jsonResponse({ id: 15 }, { status: 201 }),
    );
    const client = createApiClient({
      baseUrl: 'http://localhost:8080/api',
      getToken: () => 'secret-jwt',
      onUnauthorized: vi.fn(),
      fetchImpl: fetchMock,
    });

    const created = await client.post<{ id: number }>('/reservations', {
      showtimeId: 7,
      seatIds: [1, 3],
    });

    expect(created).toEqual({ id: 15 });
    const [, init] = fetchMock.mock.calls[0];
    const headers = new Headers(init?.headers);
    expect(headers.get('Authorization')).toBe('Bearer secret-jwt');
    expect(headers.get('Content-Type')).toBe('application/json');
    expect(init?.body).toBe(JSON.stringify({ showtimeId: 7, seatIds: [1, 3] }));
  });

  it('returns undefined on 204 No Content', async () => {
    const fetchMock = vi.fn<typeof fetch>().mockResolvedValue(
      new Response(null, { status: 204 }),
    );
    const client = createApiClient({
      baseUrl: 'http://localhost:8080/api',
      getToken: () => 'secret-jwt',
      onUnauthorized: vi.fn(),
      fetchImpl: fetchMock,
    });

    const result = await client.delete<void>('/reservations/15');
    expect(result).toBeUndefined();
  });

  it('parses the standard API error envelope and Retry-After header', async () => {
    const fetchMock = vi.fn<typeof fetch>().mockResolvedValue(
      jsonResponse(
        { error: { code: 'RATE_LIMITED', message: 'Too many attempts' } },
        { status: 429, headers: { 'Retry-After': '12' } },
      ),
    );
    const client = createApiClient({
      baseUrl: 'http://localhost:8080/api',
      getToken: () => null,
      onUnauthorized: vi.fn(),
      fetchImpl: fetchMock,
    });

    await expect(client.post('/auth/login', { email: 'a@b.com', password: 'x' })).rejects.toSatisfy(
      (err: unknown) =>
        err instanceof ApiError &&
        err.status === 429 &&
        err.code === 'RATE_LIMITED' &&
        err.message === 'Too many attempts' &&
        err.retryAfterSeconds === 12,
    );
  });

  it('invokes onUnauthorized on 401 when a token was sent', async () => {
    const onUnauthorized = vi.fn();
    const fetchMock = vi.fn<typeof fetch>().mockResolvedValue(
      jsonResponse(
        { error: { code: 'UNAUTHORIZED', message: 'Token expired' } },
        { status: 401 },
      ),
    );
    const client = createApiClient({
      baseUrl: 'http://localhost:8080/api',
      getToken: () => 'expired-token',
      onUnauthorized,
      fetchImpl: fetchMock,
    });

    await expect(client.get('/auth/me')).rejects.toBeInstanceOf(ApiError);
    expect(onUnauthorized).toHaveBeenCalledTimes(1);
  });

  it('does not invoke onUnauthorized on 401 when no token was sent or skipAuthHandling is true', async () => {
    const onUnauthorized = vi.fn();
    const fetchMock = vi.fn<typeof fetch>().mockImplementation(async () =>
      jsonResponse(
        { error: { code: 'UNAUTHORIZED', message: 'Invalid credentials' } },
        { status: 401 },
      ),
    );
    const client = createApiClient({
      baseUrl: 'http://localhost:8080/api',
      getToken: () => null,
      onUnauthorized,
      fetchImpl: fetchMock,
    });

    await expect(
      client.post('/auth/login', { email: 'a@b.com', password: 'wrong' }),
    ).rejects.toBeInstanceOf(ApiError);
    expect(onUnauthorized).not.toHaveBeenCalled();

    const authedClient = createApiClient({
      baseUrl: 'http://localhost:8080/api',
      getToken: () => 'some-token',
      onUnauthorized,
      fetchImpl: fetchMock,
    });
    await expect(
      authedClient.request('/auth/login', {
        method: 'POST',
        body: { email: 'a@b.com', password: 'wrong' },
        skipAuthHandling: true,
      }),
    ).rejects.toBeInstanceOf(ApiError);
    expect(onUnauthorized).not.toHaveBeenCalled();
  });

  it('handles non-JSON error responses gracefully', async () => {
    const fetchMock = vi.fn<typeof fetch>().mockResolvedValue(
      new Response('Bad Gateway', { status: 502, statusText: 'Bad Gateway' }),
    );
    const client = createApiClient({
      baseUrl: 'http://localhost:8080/api',
      getToken: () => null,
      onUnauthorized: vi.fn(),
      fetchImpl: fetchMock,
    });

    await expect(client.get('/movies')).rejects.toMatchObject({
      status: 502,
      code: 'HTTP_ERROR',
    });
  });
});
