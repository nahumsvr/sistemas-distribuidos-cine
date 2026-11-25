import { useQuery } from '@tanstack/react-query';
import { movieKeys, moviesApi } from './api';

export function useMoviesQuery() {
  return useQuery({
    queryKey: movieKeys.all,
    queryFn: ({ signal }) => moviesApi.list(signal),
  });
}

export function useMovieQuery(movieId: number | null) {
  return useQuery({
    queryKey: movieId !== null ? movieKeys.detail(movieId) : ['movies', 'invalid'],
    queryFn: ({ signal }) => moviesApi.getById(movieId as number, signal),
    enabled: movieId !== null,
  });
}

export function useMovieShowtimesQuery(movieId: number | null) {
  return useQuery({
    queryKey: movieId !== null ? movieKeys.showtimes(movieId) : ['movies', 'invalid', 'showtimes'],
    queryFn: ({ signal }) => moviesApi.getShowtimes(movieId as number, signal),
    enabled: movieId !== null,
  });
}
