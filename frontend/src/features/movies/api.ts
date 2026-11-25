import { apiClient } from '../../shared/api/client';
import type { Movie, Showtime } from './types';

export const movieKeys = {
  all: ['movies'] as const,
  detail: (movieId: number) => ['movies', movieId] as const,
  showtimes: (movieId: number) => ['movies', movieId, 'showtimes'] as const,
};

export const moviesApi = {
  list(signal?: AbortSignal): Promise<Movie[]> {
    return apiClient.get<Movie[]>('/movies', { signal });
  },

  getById(movieId: number, signal?: AbortSignal): Promise<Movie> {
    return apiClient.get<Movie>(`/movies/${movieId}`, { signal });
  },

  getShowtimes(movieId: number, signal?: AbortSignal): Promise<Showtime[]> {
    return apiClient.get<Showtime[]>(`/movies/${movieId}/showtimes`, { signal });
  },
};
