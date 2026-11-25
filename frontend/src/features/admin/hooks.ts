import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { movieKeys } from '../movies/api';
import { adminApi, adminKeys } from './api';
import type { MoviePayload, ShowtimePayload } from './types';

export function useAuditoriumsQuery() {
  return useQuery({
    queryKey: adminKeys.auditoriums,
    queryFn: ({ signal }) => adminApi.listAuditoriums(signal),
  });
}

export function useCreateMovieMutation() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (payload: MoviePayload) => adminApi.createMovie(payload),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: movieKeys.all });
    },
  });
}

export function useUpdateMovieMutation() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: ({ movieId, payload }: { movieId: number; payload: MoviePayload }) =>
      adminApi.updateMovie(movieId, payload),
    onSuccess: (_data, variables) => {
      void queryClient.invalidateQueries({ queryKey: movieKeys.all });
      void queryClient.invalidateQueries({ queryKey: movieKeys.detail(variables.movieId) });
    },
  });
}

export function useDeleteMovieMutation() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (movieId: number) => adminApi.deleteMovie(movieId),
    onSuccess: (_data, movieId) => {
      void queryClient.invalidateQueries({ queryKey: movieKeys.all });
      void queryClient.invalidateQueries({ queryKey: movieKeys.detail(movieId) });
      void queryClient.invalidateQueries({ queryKey: movieKeys.showtimes(movieId) });
    },
  });
}

export function useCreateShowtimeMutation() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (payload: ShowtimePayload) => adminApi.createShowtime(payload),
    onSuccess: (data) => {
      void queryClient.invalidateQueries({ queryKey: movieKeys.showtimes(data.movieId) });
    },
  });
}

export function useUpdateShowtimeMutation() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: ({
      showtimeId,
      payload,
    }: {
      showtimeId: number;
      payload: ShowtimePayload;
    }) => adminApi.updateShowtime(showtimeId, payload),
    onSuccess: (data) => {
      void queryClient.invalidateQueries({ queryKey: movieKeys.showtimes(data.movieId) });
    },
  });
}

export function useDeleteShowtimeMutation() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: ({ showtimeId }: { showtimeId: number; movieId: number }) =>
      adminApi.deleteShowtime(showtimeId),
    onSuccess: (_data, variables) => {
      void queryClient.invalidateQueries({
        queryKey: movieKeys.showtimes(variables.movieId),
      });
    },
  });
}
