import { QueryClientProvider } from '@tanstack/react-query';
import { render, screen, within } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { createMemoryRouter, RouterProvider } from 'react-router-dom';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { createQueryClient } from '../../shared/api/queryClient';
import { tokenStorage } from '../../shared/auth/tokenStorage';
import { appRoutes } from '../../shared/router';
import type { User } from '../auth/types';
import type { Movie, Showtime } from '../movies/types';
import type { Auditorium } from './types';

const ADMIN_USER: User = {
  id: 1,
  name: 'Admin Cine',
  email: 'admin@cine.local',
  role: 'admin',
  createdAt: '2026-11-24T17:00:00Z',
};

const AUDITORIUMS: Auditorium[] = [
  { id: 1, name: 'Sala 1', rows: 8, cols: 10 },
  { id: 2, name: 'Sala 2', rows: 6, cols: 8 },
];

const INITIAL_MOVIES: Movie[] = [
  {
    id: 1,
    title: 'Dune: Part Two',
    synopsis: 'Paul Atreides unites with the Fremen.',
    durationMin: 166,
    rating: 'B',
    posterUrl: 'https://cdn.example.com/dune.jpg',
  },
];

const INITIAL_SHOWTIMES: Showtime[] = [
  {
    id: 10,
    movieId: 1,
    auditorium: 'Sala 1',
    startsAt: '2027-01-15T19:00:00.000Z',
    priceCents: 8500,
  },
];

function jsonResponse(body: unknown, status = 200): Response {
  return new Response(JSON.stringify(body), {
    status,
    headers: { 'Content-Type': 'application/json' },
  });
}

function renderAdminApp() {
  tokenStorage.set('admin-jwt-token');
  const queryClient = createQueryClient();
  const router = createMemoryRouter(appRoutes, {
    initialEntries: ['/admin'],
  });
  return render(
    <QueryClientProvider client={queryClient}>
      <RouterProvider router={router} />
    </QueryClientProvider>,
  );
}

describe('Admin movies and showtimes CRUD', () => {
  beforeEach(() => {
    window.localStorage.clear();
  });

  afterEach(() => {
    vi.restoreAllMocks();
  });

  it('validates and creates, edits, and deletes movies including 409 conflict handling', async () => {
    let movies = [...INITIAL_MOVIES];
    let nextMovieId = 2;
    let deleteBlocked = true;

    vi.spyOn(globalThis, 'fetch').mockImplementation(async (input, init) => {
      const url = String(input);
      const method = init?.method ?? 'GET';

      if (url.endsWith('/api/auth/me')) {
        return jsonResponse({ user: ADMIN_USER });
      }
      if (url.endsWith('/api/admin/auditoriums')) {
        return jsonResponse(AUDITORIUMS);
      }
      if (url.endsWith('/api/movies') && method === 'GET') {
        return jsonResponse(movies);
      }
      if (url.match(/\/api\/movies\/\d+\/showtimes$/) && method === 'GET') {
        return jsonResponse(INITIAL_SHOWTIMES);
      }
      if (url.endsWith('/api/admin/movies') && method === 'POST') {
        const payload = JSON.parse(String(init?.body)) as Omit<Movie, 'id'>;
        const created: Movie = { id: nextMovieId++, ...payload };
        movies = [...movies, created];
        return jsonResponse(created, 201);
      }
      if (url.endsWith('/api/admin/movies/2') && method === 'PUT') {
        const payload = JSON.parse(String(init?.body)) as Omit<Movie, 'id'>;
        const updated: Movie = { id: 2, ...payload };
        movies = movies.map((m) => (m.id === 2 ? updated : m));
        return jsonResponse(updated);
      }
      if (url.endsWith('/api/admin/movies/2') && method === 'DELETE') {
        if (deleteBlocked) {
          return jsonResponse(
            {
              error: {
                code: 'HAS_ACTIVE_RESERVATIONS',
                message: 'Movie has active reservations',
              },
            },
            409,
          );
        }
        movies = movies.filter((m) => m.id !== 2);
        return new Response(null, { status: 204 });
      }
      return jsonResponse({ error: { code: 'NOT_FOUND', message: 'Not found' } }, 404);
    });

    const user = userEvent.setup();
    renderAdminApp();

    expect(await screen.findByText('Dune: Part Two')).toBeInTheDocument();

    // Open create form and trigger validation errors.
    await user.click(screen.getByRole('button', { name: 'Nueva película' }));
    await user.click(screen.getByRole('button', { name: 'Guardar película' }));

    const titleInput = screen.getByLabelText(/Título/);
    expect(titleInput).toHaveFocus();
    expect(titleInput).toHaveAttribute('aria-invalid', 'true');

    // Fill valid movie and submit.
    await user.type(titleInput, 'Interestelar');
    await user.type(screen.getByLabelText(/Duración/), '169');
    await user.type(screen.getByLabelText('Clasificación'), 'B');
    await user.type(screen.getByLabelText('URL del póster'), 'https://cdn.example.com/interstellar.jpg');
    await user.click(screen.getByRole('button', { name: 'Guardar película' }));

    expect(await screen.findByText('Película "Interestelar" creada.')).toBeInTheDocument();
    expect(screen.getByText('Interestelar')).toBeInTheDocument();

    // Edit the created movie.
    await user.click(screen.getByRole('button', { name: 'Editar Interestelar' }));
    const editTitleInput = screen.getByLabelText(/Título/);
    await user.clear(editTitleInput);
    await user.type(editTitleInput, 'Interstellar (IMAX)');
    await user.click(screen.getByRole('button', { name: 'Actualizar película' }));

    expect(
      await screen.findByText('Película "Interstellar (IMAX)" actualizada.'),
    ).toBeInTheDocument();

    // Attempt delete when 409 HAS_ACTIVE_RESERVATIONS occurs.
    await user.click(screen.getByRole('button', { name: 'Eliminar Interstellar (IMAX)' }));
    const dialog = screen.getByRole('dialog');
    await user.click(within(dialog).getByRole('button', { name: 'Eliminar' }));

    expect(await screen.findByRole('alert')).toHaveTextContent(
      'existen reservas activas asociadas',
    );
    expect(screen.getByText('Interstellar (IMAX)')).toBeInTheDocument();

    // Retry delete when no active reservations exist.
    deleteBlocked = false;
    await user.click(screen.getByRole('button', { name: 'Eliminar Interstellar (IMAX)' }));
    const dialogRetry = screen.getByRole('dialog');
    await user.click(within(dialogRetry).getByRole('button', { name: 'Eliminar' }));

    expect(
      await screen.findByText('Película "Interstellar (IMAX)" eliminada.'),
    ).toBeInTheDocument();
    expect(screen.queryByText('Interstellar (IMAX)')).not.toBeInTheDocument();
  });

  it('creates, edits, and deletes showtimes for the selected movie', async () => {
    let showtimes = [...INITIAL_SHOWTIMES];
    let nextShowtimeId = 11;

    vi.spyOn(globalThis, 'fetch').mockImplementation(async (input, init) => {
      const url = String(input);
      const method = init?.method ?? 'GET';

      if (url.endsWith('/api/auth/me')) {
        return jsonResponse({ user: ADMIN_USER });
      }
      if (url.endsWith('/api/admin/auditoriums')) {
        return jsonResponse(AUDITORIUMS);
      }
      if (url.endsWith('/api/movies') && method === 'GET') {
        return jsonResponse(INITIAL_MOVIES);
      }
      if (url.endsWith('/api/movies/1/showtimes') && method === 'GET') {
        return jsonResponse(showtimes);
      }
      if (url.endsWith('/api/admin/showtimes') && method === 'POST') {
        const payload = JSON.parse(String(init?.body)) as {
          movieId: number;
          auditoriumId: number;
          startsAt: string;
          priceCents: number;
        };
        const auditoriumName =
          AUDITORIUMS.find((a) => a.id === payload.auditoriumId)?.name ?? 'Sala 1';
        const created: Showtime = {
          id: nextShowtimeId++,
          movieId: payload.movieId,
          auditorium: auditoriumName,
          startsAt: payload.startsAt,
          priceCents: payload.priceCents,
        };
        showtimes = [...showtimes, created];
        return jsonResponse({ ...created, auditoriumId: payload.auditoriumId }, 201);
      }
      if (url.endsWith('/api/admin/showtimes/11') && method === 'PUT') {
        const payload = JSON.parse(String(init?.body)) as {
          movieId: number;
          auditoriumId: number;
          startsAt: string;
          priceCents: number;
        };
        const auditoriumName =
          AUDITORIUMS.find((a) => a.id === payload.auditoriumId)?.name ?? 'Sala 1';
        const updated: Showtime = {
          id: 11,
          movieId: payload.movieId,
          auditorium: auditoriumName,
          startsAt: payload.startsAt,
          priceCents: payload.priceCents,
        };
        showtimes = showtimes.map((s) => (s.id === 11 ? updated : s));
        return jsonResponse({ ...updated, auditoriumId: payload.auditoriumId });
      }
      if (url.endsWith('/api/admin/showtimes/11') && method === 'DELETE') {
        showtimes = showtimes.filter((s) => s.id !== 11);
        return new Response(null, { status: 204 });
      }
      return jsonResponse({ error: { code: 'NOT_FOUND', message: 'Not found' } }, 404);
    });

    const user = userEvent.setup();
    renderAdminApp();

    expect(
      await screen.findByRole('heading', { level: 2, name: 'Funciones de Dune: Part Two' }),
    ).toBeInTheDocument();

    // Open create showtime form.
    await user.click(screen.getByRole('button', { name: 'Nueva función' }));
    await user.selectOptions(screen.getByLabelText(/Sala/), '2');
    await user.type(screen.getByLabelText(/Fecha y hora/), '2027-02-10T20:30');
    const priceInput = screen.getByLabelText(/Precio/);
    await user.clear(priceInput);
    await user.type(priceInput, '95');
    await user.click(screen.getByRole('button', { name: 'Guardar función' }));

    expect(await screen.findByText('Función creada.')).toBeInTheDocument();
    expect(screen.getByText(/Sala 2/)).toBeInTheDocument();

    // Edit the created showtime.
    const editButtons = screen.getAllByRole('button', { name: /Editar función/ });
    await user.click(editButtons[1]);
    const editPriceInput = screen.getByLabelText(/Precio/);
    await user.clear(editPriceInput);
    await user.type(editPriceInput, '110');
    await user.click(screen.getByRole('button', { name: 'Actualizar función' }));

    expect(await screen.findByText('Función actualizada.')).toBeInTheDocument();
    expect(screen.getByText(/110\.00/)).toBeInTheDocument();

    // Delete the created showtime.
    const deleteButtons = screen.getAllByRole('button', { name: /Eliminar función/ });
    await user.click(deleteButtons[1]);
    const dialog = screen.getByRole('dialog');
    await user.click(within(dialog).getByRole('button', { name: 'Eliminar' }));

    expect(await screen.findByText('Función eliminada.')).toBeInTheDocument();
    expect(screen.queryByText(/Sala 2/)).not.toBeInTheDocument();
  });
});
