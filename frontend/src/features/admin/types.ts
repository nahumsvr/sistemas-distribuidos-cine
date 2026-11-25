export interface Auditorium {
  id: number;
  name: string;
  rows: number;
  cols: number;
}

export interface MoviePayload {
  title: string;
  synopsis: string;
  durationMin: number;
  rating: string;
  posterUrl: string;
}

export interface ShowtimePayload {
  movieId: number;
  auditoriumId: number;
  startsAt: string;
  priceCents: number;
}

export interface AdminShowtime {
  id: number;
  movieId: number;
  auditoriumId: number;
  auditorium: string;
  startsAt: string;
  priceCents: number;
}
