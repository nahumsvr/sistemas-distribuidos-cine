import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { reservationKeys, showtimeKeys } from '../showtimes/api';
import { reservationsApi } from './api';

export function useReservationsQuery() {
  return useQuery({
    queryKey: reservationKeys.all,
    queryFn: ({ signal }) => reservationsApi.list(signal),
    staleTime: 10_000,
  });
}

export function useUpdateReservationSeatsMutation() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: ({
      reservationId,
      seatIds,
    }: {
      reservationId: number;
      showtimeId: number;
      seatIds: number[];
    }) => reservationsApi.updateSeats(reservationId, seatIds),
    onSuccess: (_data, variables) => {
      void queryClient.invalidateQueries({ queryKey: reservationKeys.all });
      void queryClient.invalidateQueries({
        queryKey: showtimeKeys.seats(variables.showtimeId),
      });
    },
  });
}

export function useCancelReservationMutation() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: ({
      reservationId,
    }: {
      reservationId: number;
      showtimeId: number;
    }) => reservationsApi.cancel(reservationId),
    onSuccess: (_data, variables) => {
      void queryClient.invalidateQueries({ queryKey: reservationKeys.all });
      void queryClient.invalidateQueries({
        queryKey: showtimeKeys.seats(variables.showtimeId),
      });
    },
  });
}
