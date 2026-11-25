import { QueryClientProvider } from '@tanstack/react-query';
import { act, render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { createMemoryRouter, RouterProvider } from 'react-router-dom';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { apiClient } from '../../shared/api/client';
import { createQueryClient } from '../../shared/api/queryClient';
import { tokenStorage } from '../../shared/auth/tokenStorage';
import { appRoutes } from '../../shared/router';
import type { User } from './types';

const USER_ANA: User = {
  id: 2,
  name: 'Ana García',
  email: 'ana@cine.local',
  role: 'user',
  createdAt: '2026-11-24T17:23:06Z',
};

const USER_ADMIN: User = {
  id: 1,
  name: 'Admin Cine',
  email: 'admin@cine.local',
  role: 'admin',
  createdAt: '2026-11-24T17:00:00Z',
};

function jsonResponse(body: unknown, init?: ResponseInit): Response {
  return new Response(JSON.stringify(body), {
    status: init?.status ?? 200,
    headers: {
      'Content-Type': 'application/json',
      ...(init?.headers ?? {}),
    },
  });
}

function renderApp(initialEntry: string) {
  const queryClient = createQueryClient();
  const router = createMemoryRouter(appRoutes, {
    initialEntries: [initialEntry],
  });
  return {
    router,
    ...render(
      <QueryClientProvider client={queryClient}>
        <RouterProvider router={router} />
      </QueryClientProvider>,
    ),
  };
}

describe('Auth feature flows', () => {
  beforeEach(() => {
    window.localStorage.clear();
  });

  afterEach(() => {
    vi.restoreAllMocks();
  });

  it('redirects unauthenticated users from /reservations to /login and returns after login', async () => {
    const fetchMock = vi.spyOn(globalThis, 'fetch').mockResolvedValue(
      jsonResponse({ token: 'jwt-valid-token', user: USER_ANA }),
    );
    const user = userEvent.setup();

    renderApp('/reservations');

    expect(
      await screen.findByRole('heading', { level: 1, name: 'Iniciar sesión' }),
    ).toBeInTheDocument();

    const emailInput = screen.getByLabelText('Correo electrónico');
    const passwordInput = screen.getByLabelText('Contraseña');
    expect(emailInput).toHaveAttribute('autocomplete', 'username');
    expect(passwordInput).toHaveAttribute('autocomplete', 'current-password');

    // Show/hide password toggle works accessibly.
    const toggleBtn = screen.getByRole('button', { name: 'Mostrar contraseña' });
    expect(passwordInput).toHaveAttribute('type', 'password');
    await user.click(toggleBtn);
    expect(passwordInput).toHaveAttribute('type', 'text');
    expect(screen.getByRole('button', { name: 'Ocultar contraseña' })).toHaveAttribute(
      'aria-pressed',
      'true',
    );

    await user.type(emailInput, 'ana@cine.local');
    await user.type(passwordInput, 'User1234!');
    await user.click(screen.getByRole('button', { name: 'Iniciar sesión' }));

    expect(
      await screen.findByRole('heading', { level: 1, name: 'Mis reservas' }),
    ).toBeInTheDocument();
    expect(tokenStorage.get()).toBe('jwt-valid-token');
    expect(screen.getByText('Ana García')).toBeInTheDocument();
    expect(screen.getByRole('button', { name: 'Cerrar sesión' })).toBeInTheDocument();
    expect(fetchMock).toHaveBeenCalledTimes(1);
  });

  it('validates login fields after interaction and displays API 401/429 errors without redirecting', async () => {
    const fetchMock = vi
      .spyOn(globalThis, 'fetch')
      .mockResolvedValueOnce(
        jsonResponse(
          { error: { code: 'UNAUTHORIZED', message: 'Invalid credentials' } },
          { status: 401 },
        ),
      )
      .mockResolvedValueOnce(
        jsonResponse(
          { error: { code: 'RATE_LIMITED', message: 'Too many requests' } },
          { status: 429, headers: { 'Retry-After': '15' } },
        ),
      );
    const user = userEvent.setup();

    renderApp('/login');
    await screen.findByRole('heading', { level: 1, name: 'Iniciar sesión' });

    // No validation errors before interaction.
    expect(screen.queryByText(/Error:/)).not.toBeInTheDocument();

    // Empty submit focuses the email field and shows field errors.
    await user.click(screen.getByRole('button', { name: 'Iniciar sesión' }));
    const emailInput = screen.getByLabelText('Correo electrónico');
    expect(emailInput).toHaveFocus();
    expect(emailInput).toHaveAttribute('aria-invalid', 'true');
    expect(fetchMock).not.toHaveBeenCalled();

    await user.type(emailInput, 'ana@cine.local');
    await user.type(screen.getByLabelText('Contraseña'), 'WrongPass1!');
    await user.click(screen.getByRole('button', { name: 'Iniciar sesión' }));

    expect(await screen.findByRole('alert')).toHaveTextContent(
      'Correo electrónico o contraseña incorrectos.',
    );

    await user.click(screen.getByRole('button', { name: 'Iniciar sesión' }));
    expect(await screen.findByRole('alert')).toHaveTextContent('15 segundos');
    expect(tokenStorage.get()).toBeNull();
  });

  it('registers a new account, validates byte length, and handles 409 EMAIL_TAKEN', async () => {
    vi.spyOn(globalThis, 'fetch')
      .mockResolvedValueOnce(
        jsonResponse(
          { error: { code: 'EMAIL_TAKEN', message: 'Email already registered' } },
          { status: 409 },
        ),
      )
      .mockResolvedValueOnce(
        jsonResponse({ token: 'jwt-new-user', user: USER_ANA }, { status: 201 }),
      );
    const user = userEvent.setup();

    renderApp('/register');
    await screen.findByRole('heading', { level: 1, name: 'Crear cuenta' });

    const nameInput = screen.getByLabelText('Nombre completo');
    const emailInput = screen.getByLabelText('Correo electrónico');
    const passwordInput = screen.getByLabelText('Contraseña');
    expect(nameInput).toHaveAttribute('autocomplete', 'name');
    expect(emailInput).toHaveAttribute('autocomplete', 'username');
    expect(passwordInput).toHaveAttribute('autocomplete', 'new-password');

    await user.type(nameInput, 'Ana García');
    await user.type(emailInput, 'ana@cine.local');
    await user.type(passwordInput, 'short');
    await user.tab();
    expect(passwordInput).toHaveAttribute('aria-invalid', 'true');

    await user.clear(passwordInput);
    await user.type(passwordInput, 'SecurePass1!');
    await user.click(screen.getByRole('button', { name: 'Crear cuenta' }));

    expect(await screen.findByRole('alert')).toHaveTextContent(
      'Ya existe una cuenta registrada con este correo electrónico.',
    );

    await user.click(screen.getByRole('button', { name: 'Crear cuenta' }));
    expect(await screen.findByRole('heading', { level: 1, name: 'Cartelera' })).toBeInTheDocument();
    expect(tokenStorage.get()).toBe('jwt-new-user');
  });

  it('hydrates session from stored token, enforces admin role on /admin, and logs out', async () => {
    tokenStorage.set('jwt-stored-user');
    vi.spyOn(globalThis, 'fetch').mockResolvedValue(jsonResponse({ user: USER_ANA }));
    const user = userEvent.setup();

    renderApp('/admin');

    expect(
      await screen.findByRole('heading', { level: 1, name: 'Acceso restringido' }),
    ).toBeInTheDocument();
    expect(screen.queryByRole('link', { name: 'Administración' })).not.toBeInTheDocument();

    await user.click(screen.getByRole('button', { name: 'Cerrar sesión' }));

    expect(await screen.findByRole('heading', { level: 1, name: 'Cartelera' })).toBeInTheDocument();
    expect(tokenStorage.get()).toBeNull();
    expect(screen.getByRole('link', { name: 'Iniciar sesión' })).toBeInTheDocument();
  });

  it('allows admin users into /admin and displays the admin navigation link', async () => {
    tokenStorage.set('jwt-admin-token');
    vi.spyOn(globalThis, 'fetch').mockResolvedValue(jsonResponse({ user: USER_ADMIN }));

    renderApp('/admin');

    expect(
      await screen.findByRole('heading', { level: 1, name: 'Administración' }),
    ).toBeInTheDocument();
    expect(screen.getByRole('link', { name: 'Administración' })).toHaveAttribute(
      'aria-current',
      'page',
    );
  });

  it('clears token and redirects to /login with expiration notice when /auth/me returns 401', async () => {
    tokenStorage.set('jwt-expired');
    vi.spyOn(globalThis, 'fetch').mockResolvedValue(
      jsonResponse({ error: { code: 'UNAUTHORIZED', message: 'Token expired' } }, { status: 401 }),
    );

    renderApp('/reservations');

    expect(
      await screen.findByRole('heading', { level: 1, name: 'Iniciar sesión' }),
    ).toBeInTheDocument();
    expect(screen.getByRole('status')).toHaveTextContent(
      'Tu sesión expiró. Inicia sesión nuevamente para continuar.',
    );
    expect(tokenStorage.get()).toBeNull();
  });

  it('clears token and redirects to /login when any authenticated API call returns 401', async () => {
    tokenStorage.set('jwt-active');
    vi.spyOn(globalThis, 'fetch')
      .mockResolvedValueOnce(jsonResponse({ user: USER_ANA }))
      .mockResolvedValueOnce(
        jsonResponse(
          { error: { code: 'UNAUTHORIZED', message: 'Token expired' } },
          { status: 401 },
        ),
      );

    renderApp('/reservations');

    expect(
      await screen.findByRole('heading', { level: 1, name: 'Mis reservas' }),
    ).toBeInTheDocument();

    await act(async () => {
      await expect(apiClient.get('/reservations')).rejects.toThrow();
    });

    await waitFor(() => {
      expect(screen.getByRole('heading', { level: 1, name: 'Iniciar sesión' })).toBeInTheDocument();
    });
    expect(screen.getByRole('status')).toHaveTextContent('Tu sesión expiró');
    expect(tokenStorage.get()).toBeNull();
  });
});
