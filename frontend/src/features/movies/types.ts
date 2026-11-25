export interface Movie {
  id: number;
  title: string;
  synopsis: string;
  durationMin: number;
  rating: string;
  posterUrl: string;
}

export interface Showtime {
  id: number;
  movieId: number;
  auditorium: string;
  startsAt: string;
  priceCents: number;
}
