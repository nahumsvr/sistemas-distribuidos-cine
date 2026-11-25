import type { Showtime } from '../movies/types';

export interface Seat {
  id: number;
  row: string;
  number: number;
  available: boolean;
}

export interface ShowtimeSeatsResponse {
  showtime: Showtime;
  seats: Seat[];
}

export interface CreateReservationInput {
  showtimeId: number;
  seatIds: number[];
}

export interface ReservedSeat {
  id: number;
  row: string;
  number: number;
}

export interface Reservation {
  id: number;
  status: 'active' | 'cancelled';
  showtime: {
    id: number;
    movieTitle: string;
    startsAt: string;
  };
  seats: ReservedSeat[];
  totalCents: number;
  createdAt: string;
}

export const MAX_SEATS_PER_RESERVATION = 10;
