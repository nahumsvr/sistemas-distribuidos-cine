import { QueryClientProvider } from '@tanstack/react-query';
import { render, screen, waitFor, within } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { createMemoryRouter, RouterProvider } from 'react-router-dom';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { createQueryClient } from '../../shared/api/queryClient';
import { tokenStorage } from '../../shared/auth/tokenStorage';
import { appRoutes } from '../../shared/router';
import type { Reservation, Seat, ShowtimeSeatsResponse } from '../showtimes/types';

function makeSeats(count: number, occupiedIds: number[] = []): Seat[] {
  const occupied = new Set(occupiedIds);
  return Array.from({ length: count }, (_, idx) => {
    const id = idx + 1;
    const row = id <= 6 ? 'A' : 'B';
    const number = id <= 6 ? id : id - 6;
    return { id, row, number, available: !occupied.has(id) };
  });
}

const USER_ANA = {
  id: 2,
  name: 'Ana García',
  email: 'ana@cine.local',
  role: 'user',
  createdAt: '2026-11-24T17:23:06Z',
};

const ACTIVE_FUTURE_RESERVATION: Reservation = {
  id: 15,
  status: 'active',
  showtime: {
    id: 7,
    movieTitle: 'Dune: Part Two',
    startsAt: '2099-12-05T20:00:00Z',
  },
  seats: [
    { id: 1, row: 'A', number: 1 },
    { id: 3, row: 'A', number: 3 },
  ],
  totalCents: 16000,
  createdAt: '2026-11-25T10:00:00Z',
};

const ACTIVE_STARTED_RESERVATION: Reservation = {
  id: 12,
  status: 'active',
  showtime: {
    id: 5,
    movieTitle: 'Interstellar',
    startsAt: '2020-01-01T20:00:00Z',
  },
  seats: [{ id: 4, row: 'A', number: 4 }],
  totalCents: 8000,
  createdAt: '2020-01-01T18:00:00Z',
};

const CANCELLED_RESERVATION: Reservation = {
  id: 10,
  status: 'cancelled',
  showtime: {
    id: 6,
    movieTitle: 'Arrival',
    startsAt: '2099-12-10T18:00:00Z',
  },
  seats: [],
  totalCents: 0,
  createdAt: '2026-11-24T12:00:00Z',
};

const SHOWTIME_SEATS: ShowtimeSeatsResponse = {
  showtime: {
    id: 7,
    movieId: 1,
    auditorium: 'Sala 1',
    startsAt: '2099-12-05T20:00:00Z',
    priceCents: 8000,
  },
  // Seats 1 and 3 belong to reservation #15; seat 2 belongs to someone else.
  seats: makeSeats(12, [1, 2, 3]),
};

function jsonResponse(body: unknown, status = 200): Response {
  return new Response(JSON.stringify(body), {
    status,
    headers: { 'Content-Type': 'application/json' },
  });
}

function renderReservationsApp() {
  const queryClient = createQueryClient();
  const router = createMemoryRouter(appRoutes, {
    initialEntries: ['/reservations'],
  });
  return render(
    <QueryClientProvider client={queryClient}>
      <RouterProvider router={router} />
    </QueryClientProvider>,
  );
}

describe('ReservationsPage (Mis reservas)', () => {
  beforeEach(() => {
    window.localStorage.clear();
    tokenStorage.set('jwt-valid');
  });

  afterEach(() => {
    vi.restoreAllMocks();
  });

  it('lists active, started, and cancelled reservations with appropriate actions', async () => {
    vi.spyOn(globalThis, 'fetch').mockImplementation(async (input) => {
      const url = String(input);
      if (url.endsWith('/api/auth/me')) {
        return jsonResponse({ user: USER_ANA });
      }
      if (url.endsWith('/api/reservations')) {
        return jsonResponse([
          ACTIVE_FUTURE_RESERVATION,
          ACTIVE_STARTED_RESERVATION,
          CANCELLED_RESERVATION,
        ]);
      }
      return jsonResponse({ error: { code: 'NOT_FOUND', message: 'Not found' } }, 404);
    });

    renderReservationsApp();

    expect(
      await screen.findByRole('heading', { level: 1, name: 'Mis reservas' }),
    ).toBeInTheDocument();

    const activeCard = screen.getByRole('article', { name: 'Dune: Part Two' });
    expect(within(activeCard).getByText('Activa')).toBeInTheDocument();
    expect(within(activeCard).getByText('A1, A3')).toBeInTheDocument();
    expect(
      within(activeCard).getByRole('button', { name: 'Modificar asientos' }),
    ).toBeInTheDocument();
    expect(
      within(activeCard).getByRole('button', { name: 'Cancelar reserva' }),
    ).toBeInTheDocument();

    const startedCard = screen.getByRole('article', { name: 'Interstellar' });
    expect(within(startedCard).getByText('Función iniciada')).toBeInTheDocument();
    expect(
      within(startedCard).queryByRole('button', { name: 'Modificar asientos' }),
    ).not.toBeInTheDocument();

    const cancelledCard = screen.getByRole('article', { name: 'Arrival' });
    expect(within(cancelledCard).getByText('Cancelada')).toBeInTheDocument();
    expect(within(cancelledCard).getByText('Sin asientos')).toBeInTheDocument();
    expect(
      within(cancelledCard).queryByRole('button', { name: 'Cancelar reserva' }),
    ).not.toBeInTheDocument();
  });

  it('renders an empty state when the user has no reservations', async () => {
    vi.spyOn(globalThis, 'fetch').mockImplementation(async (input) => {
      const url = String(input);
      if (url.endsWith('/api/auth/me')) {
        return jsonResponse({ user: USER_ANA });
      }
      if (url.endsWith('/api/reservations')) {
        return jsonResponse([]);
      }
      return jsonResponse({ error: { code: 'NOT_FOUND', message: 'Not found' } }, 404);
    });

    renderReservationsApp();

    expect(
      await screen.findByRole('heading', { level: 2, name: 'Todavía no tienes reservas' }),
    ).toBeInTheDocument();
    expect(screen.getByRole('link', { name: 'Ver cartelera' })).toHaveAttribute('href', '/');
  });

  it('allows modifying seats and handles 409 conflict before saving', async () => {
    let reservations = [ACTIVE_FUTURE_RESERVATION];
    let putAttempts = 0;

    vi.spyOn(globalThis, 'fetch').mockImplementation(async (input, init) => {
      const url = String(input);
      if (url.endsWith('/api/auth/me')) {
        return jsonResponse({ user: USER_ANA });
      }
      if (url.endsWith('/api/reservations') && (!init?.method || init.method === 'GET')) {
        return jsonResponse(reservations);
      }
      if (url.endsWith('/api/showtimes/7/seats')) {
        return jsonResponse(
          putAttempts === 0
            ? SHOWTIME_SEATS
            : { ...SHOWTIME_SEATS, seats: makeSeats(12, [1, 2, 3, 4]) },
        );
      }
      if (url.endsWith('/api/reservations/15') && init?.method === 'PUT') {
        putAttempts += 1;
        const body = JSON.parse(String(init.body)) as { seatIds: number[] };
        if (putAttempts === 1) {
          return jsonResponse(
            { error: { code: 'SEAT_TAKEN', message: 'Seat already taken' } },
            409,
          );
        }
        const updated: Reservation = {
          ...ACTIVE_FUTURE_RESERVATION,
          seats: body.seatIds.map((id) => ({ id, row: 'A', number: id })),
          totalCents: body.seatIds.length * 8000,
        };
        reservations = [updated];
        return jsonResponse(updated);
      }
      return jsonResponse({ error: { code: 'NOT_FOUND', message: 'Not found' } }, 404);
    });

    const user = userEvent.setup();
    renderReservationsApp();

    await user.click(await screen.findByRole('button', { name: 'Modificar asientos' }));

    const dialog = await screen.findByRole('dialog');
    // User's own seats (A1, A3) are pre-selected and enabled; A2 (other user) is disabled.
    const seatA1 = await within(dialog).findByRole('button', {
      name: 'Asiento A1, seleccionado',
    });
    expect(seatA1).toBeEnabled();
    expect(
      within(dialog).getByRole('button', { name: 'Asiento A2, ocupado' }),
    ).toBeDisabled();

    // Deselect A1 and select A4 -> triggers 409 on first save.
    await user.click(seatA1);
    await user.click(within(dialog).getByRole('button', { name: 'Asiento A4, disponible' }));
    await user.click(within(dialog).getByRole('button', { name: 'Guardar cambios' }));

    expect(await within(dialog).findByRole('alert')).toHaveTextContent(
      'Uno o más asientos seleccionados ya fueron reservados',
    );
    await waitFor(() => {
      expect(
        within(dialog).getByRole('button', { name: 'Asiento A4, ocupado' }),
      ).toBeDisabled();
    });

    // Select A5 and save successfully.
    await user.click(within(dialog).getByRole('button', { name: 'Asiento A5, disponible' }));
    await user.click(within(dialog).getByRole('button', { name: 'Guardar cambios' }));

    await waitFor(() => {
      expect(screen.queryByRole('dialog')).not.toBeInTheDocument();
    });
    expect(screen.getByRole('status')).toHaveTextContent('Reserva #15 actualizada');
    expect(screen.getByText('A3, A5')).toBeInTheDocument();
  });

  it('confirms and cancels an active reservation', async () => {
    let reservations = [ACTIVE_FUTURE_RESERVATION];

    vi.spyOn(globalThis, 'fetch').mockImplementation(async (input, init) => {
      const url = String(input);
      if (url.endsWith('/api/auth/me')) {
        return jsonResponse({ user: USER_ANA });
      }
      if (url.endsWith('/api/reservations') && (!init?.method || init.method === 'GET')) {
        return jsonResponse(reservations);
      }
      if (url.endsWith('/api/reservations/15') && init?.method === 'DELETE') {
        reservations = [
          {
            ...ACTIVE_FUTURE_RESERVATION,
            status: 'cancelled',
            seats: [],
            totalCents: 0,
          },
        ];
        return new Response(null, { status: 204 });
      }
      return jsonResponse({ error: { code: 'NOT_FOUND', message: 'Not found' } }, 404);
    });

    const user = userEvent.setup();
    renderReservationsApp();

    await user.click(await screen.findByRole('button', { name: 'Cancelar reserva' }));

    const dialog = await screen.findByRole('dialog');
    expect(
      within(dialog).getByRole('heading', { level: 1, name: '¿Cancelar reserva #15?' }),
    ).toBeInTheDocument();

    await user.click(within(dialog).getByRole('button', { name: 'Sí, cancelar reserva' }));

    await waitFor(() => {
      expect(screen.queryByRole('dialog')).not.toBeInTheDocument();
    });
    expect(screen.getByRole('status')).toHaveTextContent('Reserva #15 cancelada');
    expect(screen.getByText('Cancelada')).toBeInTheDocument();
    expect(screen.getByText('Sin asientos')).toBeInTheDocument();
  });
});
