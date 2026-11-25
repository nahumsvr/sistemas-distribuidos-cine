import { apiClient, isApiError } from '../../shared/api/client';
import type { Reservation } from '../showtimes/types';

export const reservationsApi = {
  list(signal?: AbortSignal): Promise<Reservation[]> {
    return apiClient.get<Reservation[]>('/reservations', { signal });
  },

  updateSeats(reservationId: number, seatIds: number[]): Promise<Reservation> {
    return apiClient.put<Reservation>(`/reservations/${reservationId}`, { seatIds });
  },

  cancel(reservationId: number): Promise<void> {
    return apiClient.delete(`/reservations/${reservationId}`);
  },
};

export function getReservationActionErrorMessage(
  error: unknown,
  action: 'update' | 'cancel',
): string {
  if (isApiError(error)) {
    if (error.status === 409 || error.code === 'SEAT_TAKEN') {
      return 'Uno o más asientos seleccionados ya fueron reservados. Actualizamos la disponibilidad.';
    }
    if (error.status === 422 || error.code === 'SHOWTIME_STARTED') {
      return action === 'update'
        ? 'Esta función ya comenzó y no admite cambios de asientos.'
        : 'Esta función ya comenzó y la reserva no se puede cancelar.';
    }
    if (error.status === 400 || error.code === 'VALIDATION_ERROR') {
      return action === 'update'
        ? 'Selección inválida. Elige entre 1 y 10 asientos disponibles.'
        : 'No se pudo cancelar esta reserva.';
    }
    if (error.status === 404 || error.code === 'NOT_FOUND') {
      return 'La reserva ya no existe o no está disponible.';
    }
    if (error.status === 401 || error.code === 'UNAUTHORIZED') {
      return 'Tu sesión expiró. Inicia sesión nuevamente para continuar.';
    }
    if (error.status >= 500) {
      return 'El servidor no pudo completar la operación. Intenta nuevamente en unos momentos.';
    }
  }

  return action === 'update'
    ? 'No se pudieron actualizar los asientos. Verifica tu conexión e inténtalo nuevamente.'
    : 'No se pudo cancelar la reserva. Verifica tu conexión e inténtalo nuevamente.';
}
