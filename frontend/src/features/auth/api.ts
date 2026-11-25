import { apiClient, isApiError } from '../../shared/api/client';
import type { AuthResponse, LoginInput, MeResponse, RegisterInput, User } from './types';

export const authApi = {
  login(input: LoginInput): Promise<AuthResponse> {
    return apiClient.post<AuthResponse>('/auth/login', input, {
      skipAuthHandling: true,
    });
  },

  register(input: RegisterInput): Promise<AuthResponse> {
    return apiClient.post<AuthResponse>('/auth/register', input, {
      skipAuthHandling: true,
    });
  },

  async me(signal?: AbortSignal): Promise<User> {
    const response = await apiClient.get<MeResponse>('/auth/me', { signal });
    return response.user;
  },
};

export function getAuthErrorMessage(error: unknown, mode: 'login' | 'register'): string {
  if (isApiError(error)) {
    if (error.status === 429 || error.code === 'RATE_LIMITED') {
      if (error.retryAfterSeconds !== undefined && error.retryAfterSeconds > 0) {
        return `Demasiados intentos. Intenta nuevamente en ${error.retryAfterSeconds} segundos.`;
      }
      return 'Demasiados intentos. Espera un momento antes de volver a intentarlo.';
    }

    if (mode === 'login' && (error.status === 401 || error.code === 'UNAUTHORIZED')) {
      return 'Correo electrónico o contraseña incorrectos.';
    }

    if (mode === 'register' && (error.status === 409 || error.code === 'EMAIL_TAKEN')) {
      return 'Ya existe una cuenta registrada con este correo electrónico.';
    }

    if (error.status === 400 || error.code === 'VALIDATION_ERROR') {
      return 'Revisa los datos ingresados e inténtalo nuevamente.';
    }

    if (error.status >= 500) {
      return 'El servidor no pudo procesar la solicitud. Intenta nuevamente en unos momentos.';
    }
  }

  return 'No se pudo conectar con el servidor. Verifica tu conexión e inténtalo nuevamente.';
}

/**
 * Accepts only same-origin relative paths (rejecting protocol-relative URLs like //evil.com).
 */
export function getSafeRedirectPath(from: unknown): string {
  if (typeof from === 'string' && from.startsWith('/') && !from.startsWith('//')) {
    return from;
  }
  return '/';
}
