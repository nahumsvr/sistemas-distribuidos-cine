import { QueryClientProvider } from '@tanstack/react-query';
import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { createMemoryRouter, RouterProvider } from 'react-router-dom';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { createQueryClient } from './api/queryClient';
import { appRoutes } from './router';

function renderAt(initialEntry: string) {
  const queryClient = createQueryClient();
  const router = createMemoryRouter(appRoutes, {
    initialEntries: [initialEntry],
  });
  return render(
    <QueryClientProvider client={queryClient}>
      <RouterProvider router={router} />
    </QueryClientProvider>,
  );
}

describe('App layout and router', () => {
  beforeEach(() => {
    window.localStorage.clear();
    vi.spyOn(globalThis, 'fetch').mockResolvedValue(
      new Response(JSON.stringify([]), {
        status: 200,
        headers: { 'Content-Type': 'application/json' },
      }),
    );
  });

  afterEach(() => {
    vi.restoreAllMocks();
  });

  it('renders semantic landmarks, skip link, active nav link, and Cartelera route', async () => {
    renderAt('/');

    expect(screen.getByRole('banner')).toBeInTheDocument();
    expect(screen.getByRole('navigation', { name: 'Principal' })).toBeInTheDocument();
    expect(screen.getByRole('main')).toBeInTheDocument();
    expect(screen.getByRole('contentinfo')).toBeInTheDocument();
    expect(screen.getByRole('link', { name: 'Ir al contenido principal' })).toHaveAttribute(
      'href',
      '#main-content',
    );

    expect(await screen.findByRole('heading', { level: 1, name: 'Cartelera' })).toBeInTheDocument();
    expect(document.title).toBe('Cartelera | Cine');
    expect(screen.getByRole('link', { name: 'Cartelera' })).toHaveAttribute('aria-current', 'page');
  });

  it('navigates between lazy-loaded routes and updates active indicator and title', async () => {
    const user = userEvent.setup();
    renderAt('/');

    await screen.findByRole('heading', { level: 1, name: 'Cartelera' });
    await user.click(screen.getByRole('link', { name: 'Iniciar sesión' }));

    expect(
      await screen.findByRole('heading', { level: 1, name: 'Iniciar sesión' }),
    ).toBeInTheDocument();
    expect(document.title).toBe('Iniciar sesión | Cine');
    expect(screen.getByRole('link', { name: 'Iniciar sesión' })).toHaveAttribute(
      'aria-current',
      'page',
    );
    expect(screen.getByRole('link', { name: 'Cartelera' })).not.toHaveAttribute('aria-current');
  });

  it('renders the 404 page on unknown routes', async () => {
    renderAt('/ruta-que-no-existe');

    expect(
      await screen.findByRole('heading', { level: 1, name: 'Página no encontrada' }),
    ).toBeInTheDocument();
    expect(document.title).toBe('Página no encontrada | Cine');
  });
});
