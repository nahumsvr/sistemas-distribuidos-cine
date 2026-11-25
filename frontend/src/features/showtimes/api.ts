import { apiClient, isApiError } from '../../shared/api/client';
import type { CreateReservationInput, Reservation, ShowtimeSeatsResponse } from './types';

export const showtimeKeys = {
  seats: (showtimeId: number) => ['showtimes', showtimeId, 'seats'] as const,
};

export const reservationKeys = {
  all: ['reservations'] as const,
};

export const showtimesApi = {
  getSeats(showtimeId: number, signal?: AbortSignal): Promise<ShowtimeSeatsResponse> {
    return apiClient.get<ShowtimeSeatsResponse>(`/showtimes/${showtimeId}/seats`, { signal });
  },

  createReservation(input: CreateReservationInput): Promise<Reservation> {
    return apiClient.post<Reservation>('/reservations', input);
  },
};

export function getReservationErrorMessage(error: unknown): string {
  if (isApiError(error)) {
    if (error.status === 409 || error.code === 'SEAT_TAKEN') {
      return 'Uno o más asientos seleccionados ya fueron reservados. Actualizamos la disponibilidad.';
    }
    if (error.status === 422 || error.code === 'SHOWTIME_STARTED') {
      return 'Esta función ya comenzó y no admite nuevas reservas.';
    }
    if (error.status === 400 || error.code === 'VALIDATION_ERROR') {
      return 'Selección inválida. Elige entre 1 y 10 asientos disponibles.';
    }
    if (error.status === 401 || error.code === 'UNAUTHORIZED') {
      return 'Tu sesión expiró. Inicia sesión nuevamente para confirmar tu reserva.';
    }
    if (error.status >= 500) {
      return 'El servidor no pudo procesar la reserva. Intenta nuevamente en unos momentos.';
    }
  }
  return 'No se pudo completar la reserva. Verifica tu conexión e inténtalo nuevamente.';
}
