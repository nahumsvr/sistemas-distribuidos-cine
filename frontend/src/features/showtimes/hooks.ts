import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { reservationKeys, showtimeKeys, showtimesApi } from './api';
import type { CreateReservationInput } from './types';

export function useShowtimeSeatsQuery(showtimeId: number | null) {
  return useQuery({
    queryKey: showtimeId !== null ? showtimeKeys.seats(showtimeId) : ['showtimes', 'invalid'],
    queryFn: ({ signal }) => showtimesApi.getSeats(showtimeId as number, signal),
    enabled: showtimeId !== null,
    staleTime: 10_000,
  });
}

export function useCreateReservationMutation(showtimeId: number) {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (input: CreateReservationInput) => showtimesApi.createReservation(input),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: showtimeKeys.seats(showtimeId) });
      void queryClient.invalidateQueries({ queryKey: reservationKeys.all });
    },
  });
}
