import { QueryClientProvider } from '@tanstack/react-query';
import { render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { createMemoryRouter, RouterProvider } from 'react-router-dom';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { createQueryClient } from '../../shared/api/queryClient';
import { tokenStorage } from '../../shared/auth/tokenStorage';
import { appRoutes } from '../../shared/router';
import type { Seat, ShowtimeSeatsResponse } from './types';

function makeSeats(count: number, occupiedIds: number[] = []): Seat[] {
  const occupied = new Set(occupiedIds);
  return Array.from({ length: count }, (_, idx) => {
    const id = idx + 1;
    const row = id <= 6 ? 'A' : 'B';
    const number = id <= 6 ? id : id - 6;
    return { id, row, number, available: !occupied.has(id) };
  });
}

const SHOWTIME_RESPONSE: ShowtimeSeatsResponse = {
  showtime: {
    id: 7,
    movieId: 1,
    auditorium: 'Sala 1',
    startsAt: '2026-12-05T20:00:00Z',
    priceCents: 8000,
  },
  seats: makeSeats(12, [2]),
};

const MOVIE = {
  id: 1,
  title: 'Dune: Part Two',
  synopsis: 'Paul Atreides unites with the Fremen.',
  durationMin: 166,
  rating: 'B',
  posterUrl: '',
};

const USER_ANA = {
  id: 2,
  name: 'Ana García',
  email: 'ana@cine.local',
  role: 'user',
  createdAt: '2026-11-24T17:23:06Z',
};

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

describe('Showtime seat map and reservation creation', () => {
  beforeEach(() => {
    window.localStorage.clear();
  });

  afterEach(() => {
    vi.restoreAllMocks();
  });

  it('renders available/occupied seats, enforces 10-seat cap, and prompts login for visitors', async () => {
    vi.spyOn(globalThis, 'fetch').mockImplementation(async (input) => {
      const url = String(input);
      if (url.endsWith('/api/showtimes/7/seats')) {
        return jsonResponse(SHOWTIME_RESPONSE);
      }
      if (url.endsWith('/api/movies/1')) {
        return jsonResponse(MOVIE);
      }
      return jsonResponse({ error: { code: 'NOT_FOUND', message: 'Not found' } }, 404);
    });
    const user = userEvent.setup();

    renderApp('/showtimes/7');

    expect(
      await screen.findByRole('heading', { level: 1, name: 'Dune: Part Two' }),
    ).toBeInTheDocument();

    const seatA2 = screen.getByRole('button', { name: 'Asiento A2, ocupado' });
    expect(seatA2).toBeDisabled();

    // Select 10 available seats (A1, A3..A6, B1..B5).
    const availableCodes = ['A1', 'A3', 'A4', 'A5', 'A6', 'B1', 'B2', 'B3', 'B4', 'B5'];
    for (const code of availableCodes) {
      await user.click(screen.getByRole('button', { name: `Asiento ${code}, disponible` }));
    }

    expect(screen.getByText('10 de 10')).toBeInTheDocument();
    expect(screen.getByText(/800\.00/)).toBeInTheDocument();

    // Clicking an 11th seat shows the 10-seat cap warning without selecting it.
    const seatB6 = screen.getByRole('button', { name: 'Asiento B6, disponible' });
    await user.click(seatB6);
    expect(seatB6).toHaveAttribute('aria-pressed', 'false');
    expect(screen.getByRole('status')).toHaveTextContent('máximo de 10 asientos');

    // Unauthenticated visitors get a login CTA that preserves the showtime route.
    expect(screen.getByRole('link', { name: 'Inicia sesión para reservar' })).toHaveAttribute(
      'href',
      '/login',
    );
  });

  it('submits reservation for authenticated users and refreshes seat availability', async () => {
    tokenStorage.set('jwt-valid');
    let seatsCalls = 0;

    const fetchMock = vi.spyOn(globalThis, 'fetch').mockImplementation(async (input, init) => {
      const url = String(input);
      if (url.endsWith('/api/auth/me')) {
        return jsonResponse({ user: USER_ANA });
      }
      if (url.endsWith('/api/movies/1')) {
        return jsonResponse(MOVIE);
      }
      if (url.endsWith('/api/showtimes/7/seats')) {
        seatsCalls += 1;
        return jsonResponse(
          seatsCalls === 1
            ? SHOWTIME_RESPONSE
            : { ...SHOWTIME_RESPONSE, seats: makeSeats(12, [1, 2, 3]) },
        );
      }
      if (url.endsWith('/api/reservations') && init?.method === 'POST') {
        expect(JSON.parse(String(init.body))).toEqual({ showtimeId: 7, seatIds: [1, 3] });
        return jsonResponse(
          {
            id: 15,
            status: 'active',
            showtime: { id: 7, movieTitle: 'Dune: Part Two', startsAt: '2026-12-05T20:00:00Z' },
            seats: [
              { id: 1, row: 'A', number: 1 },
              { id: 3, row: 'A', number: 3 },
            ],
            totalCents: 16000,
            createdAt: '2026-11-25T18:00:00Z',
          },
          201,
        );
      }
      return jsonResponse({ error: { code: 'NOT_FOUND', message: 'Not found' } }, 404);
    });
    const user = userEvent.setup();

    renderApp('/showtimes/7');

    await user.click(await screen.findByRole('button', { name: 'Asiento A1, disponible' }));
    await user.click(screen.getByRole('button', { name: 'Asiento A3, disponible' }));

    const confirmButton = await screen.findByRole('button', { name: 'Confirmar reserva' });
    await user.click(confirmButton);

    expect(await screen.findByRole('status')).toHaveTextContent('Reserva #15 confirmada');
    expect(screen.getByRole('link', { name: 'Ver mis reservas' })).toHaveAttribute(
      'href',
      '/reservations',
    );

    await waitFor(() => {
      expect(screen.getByRole('button', { name: 'Asiento A1, ocupado' })).toBeDisabled();
      expect(screen.getByRole('button', { name: 'Asiento A3, ocupado' })).toBeDisabled();
    });
    expect(fetchMock).toHaveBeenCalled();
  });

  it('handles 409 SEAT_TAKEN by showing conflict alert, clearing selection, and refetching seats', async () => {
    tokenStorage.set('jwt-valid');
    let seatsCalls = 0;

    vi.spyOn(globalThis, 'fetch').mockImplementation(async (input, init) => {
      const url = String(input);
      if (url.endsWith('/api/auth/me')) {
        return jsonResponse({ user: USER_ANA });
      }
      if (url.endsWith('/api/movies/1')) {
        return jsonResponse(MOVIE);
      }
      if (url.endsWith('/api/showtimes/7/seats')) {
        seatsCalls += 1;
        return jsonResponse(
          seatsCalls === 1
            ? SHOWTIME_RESPONSE
            : { ...SHOWTIME_RESPONSE, seats: makeSeats(12, [1, 2]) },
        );
      }
      if (url.endsWith('/api/reservations') && init?.method === 'POST') {
        return jsonResponse(
          { error: { code: 'SEAT_TAKEN', message: 'Seat already reserved' } },
          409,
        );
      }
      return jsonResponse({ error: { code: 'NOT_FOUND', message: 'Not found' } }, 404);
    });
    const user = userEvent.setup();

    renderApp('/showtimes/7');

    await user.click(await screen.findByRole('button', { name: 'Asiento A1, disponible' }));
    await user.click(await screen.findByRole('button', { name: 'Confirmar reserva' }));

    expect(await screen.findByRole('alert')).toHaveTextContent(
      'Uno o más asientos seleccionados ya fueron reservados',
    );
    await waitFor(() => {
      expect(screen.getByRole('button', { name: 'Asiento A1, ocupado' })).toBeDisabled();
    });
    expect(screen.getByText('0 de 10')).toBeInTheDocument();
  });

  it('shows not-found message for invalid showtime IDs', async () => {
    renderApp('/showtimes/invalid');

    expect(
      await screen.findByRole('heading', { level: 1, name: 'Función no encontrada' }),
    ).toBeInTheDocument();
  });
});
