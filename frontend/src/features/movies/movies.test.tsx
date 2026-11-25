import { QueryClientProvider } from '@tanstack/react-query';
import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { createMemoryRouter, RouterProvider } from 'react-router-dom';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { createQueryClient } from '../../shared/api/queryClient';
import { appRoutes } from '../../shared/router';
import type { Movie, Showtime } from './types';

const MOVIES: Movie[] = [
  {
    id: 1,
    title: 'Dune: Part Two',
    synopsis: 'Paul Atreides unites with the Fremen to seek revenge.',
    durationMin: 166,
    rating: 'B',
    posterUrl: 'https://cdn.example.com/dune.jpg',
  },
  {
    id: 2,
    title: 'Coco',
    synopsis: 'Miguel journeys to the Land of the Dead.',
    durationMin: 105,
    rating: 'A',
    posterUrl: '',
  },
];

const SHOWTIMES: Showtime[] = [
  {
    id: 10,
    movieId: 1,
    auditorium: 'Sala 1',
    startsAt: '2026-12-05T19:00:00Z',
    priceCents: 7000,
  },
  {
    id: 11,
    movieId: 1,
    auditorium: 'Sala 1',
    startsAt: '2026-12-05T23:00:00Z',
    priceCents: 8500,
  },
  {
    id: 12,
    movieId: 1,
    auditorium: 'Sala 2',
    startsAt: '2026-12-06T20:00:00Z',
    priceCents: 8500,
  },
];

function jsonResponse(body: unknown, status = 200): Response {
  return new Response(JSON.stringify(body), {
    status,
    headers: { 'Content-Type': 'application/json' },
  });
}

function renderApp(initialEntry: string) {
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

describe('Movies catalog and detail', () => {
  beforeEach(() => {
    window.localStorage.clear();
  });

  afterEach(() => {
    vi.restoreAllMocks();
  });

  it('lists movies, omits poster img when empty, and navigates to detail with grouped showtimes', async () => {
    vi.spyOn(globalThis, 'fetch').mockImplementation(async (input) => {
      const url = String(input);
      if (url.endsWith('/api/movies')) {
        return jsonResponse(MOVIES);
      }
      if (url.endsWith('/api/movies/1')) {
        return jsonResponse(MOVIES[0]);
      }
      if (url.endsWith('/api/movies/1/showtimes')) {
        return jsonResponse(SHOWTIMES);
      }
      return jsonResponse({ error: { code: 'NOT_FOUND', message: 'Not found' } }, 404);
    });
    const user = userEvent.setup();

    renderApp('/');

    expect(await screen.findByRole('link', { name: 'Dune: Part Two' })).toBeInTheDocument();
    expect(screen.getByRole('link', { name: 'Coco' })).toBeInTheDocument();
    expect(screen.getByAltText('Póster de Dune: Part Two')).toBeInTheDocument();
    expect(screen.queryByAltText('Póster de Coco')).not.toBeInTheDocument();

    await user.click(screen.getByRole('link', { name: 'Dune: Part Two' }));

    expect(
      await screen.findByRole('heading', { level: 1, name: 'Dune: Part Two' }),
    ).toBeInTheDocument();
    expect(document.title).toBe('Dune: Part Two | Cine');
    expect(await screen.findByRole('link', { name: /Sala 2/ })).toHaveAttribute(
      'href',
      '/showtimes/12',
    );
    expect(screen.getAllByRole('link', { name: /Sala 1/ })).toHaveLength(2);
  });

  it('renders empty state when there are no movies', async () => {
    vi.spyOn(globalThis, 'fetch').mockResolvedValue(jsonResponse([]));

    renderApp('/');

    expect(
      await screen.findByText('No hay películas en cartelera por el momento.'),
    ).toBeInTheDocument();
  });

  it('shows not-found view for invalid or unknown movie IDs', async () => {
    vi.spyOn(globalThis, 'fetch').mockResolvedValue(
      jsonResponse({ error: { code: 'NOT_FOUND', message: 'Movie not found' } }, 404),
    );

    renderApp('/movies/9999');

    expect(
      await screen.findByRole('heading', { level: 1, name: 'Película no encontrada' }),
    ).toBeInTheDocument();
    expect(screen.getByRole('link', { name: 'Volver a la cartelera' })).toHaveAttribute('href', '/');
  });
});
