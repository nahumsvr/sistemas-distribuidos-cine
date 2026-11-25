import { apiClient } from '../../shared/api/client';
import type { Movie } from '../movies/types';
import type {
  AdminShowtime,
  Auditorium,
  MoviePayload,
  ShowtimePayload,
} from './types';

export const adminKeys = {
  auditoriums: ['admin', 'auditoriums'] as const,
};

export const adminApi = {
  listAuditoriums(signal?: AbortSignal): Promise<Auditorium[]> {
    return apiClient.get<Auditorium[]>('/admin/auditoriums', { signal });
  },

  createMovie(payload: MoviePayload): Promise<Movie> {
    return apiClient.post<Movie>('/admin/movies', payload);
  },

  updateMovie(movieId: number, payload: MoviePayload): Promise<Movie> {
    return apiClient.put<Movie>(`/admin/movies/${movieId}`, payload);
  },

  deleteMovie(movieId: number): Promise<void> {
    return apiClient.delete<void>(`/admin/movies/${movieId}`);
  },

  createShowtime(payload: ShowtimePayload): Promise<AdminShowtime> {
    return apiClient.post<AdminShowtime>('/admin/showtimes', payload);
  },

  updateShowtime(showtimeId: number, payload: ShowtimePayload): Promise<AdminShowtime> {
    return apiClient.put<AdminShowtime>(`/admin/showtimes/${showtimeId}`, payload);
  },

  deleteShowtime(showtimeId: number): Promise<void> {
    return apiClient.delete<void>(`/admin/showtimes/${showtimeId}`);
  },
};
