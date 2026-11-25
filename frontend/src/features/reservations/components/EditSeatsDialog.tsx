import { useMemo, useState } from 'react';
import { isApiError } from '../../../shared/api/client';
import { formatPriceCents, formatShowtimeDateTime } from '../../../shared/format';
import { Button } from '../../../shared/ui/Button';
import { ErrorAlert } from '../../../shared/ui/ErrorAlert';
import { LoadingState } from '../../../shared/ui/LoadingState';
import { SeatMap } from '../../showtimes/components/SeatMap';
import { useShowtimeSeatsQuery } from '../../showtimes/hooks';
import {
  MAX_SEATS_PER_RESERVATION,
  type Reservation,
  type Seat,
} from '../../showtimes/types';
import { getReservationActionErrorMessage } from '../api';
import { useUpdateReservationSeatsMutation } from '../hooks';
import { ModalDialog } from './ModalDialog';

export interface EditSeatsDialogProps {
  reservation: Reservation | null;
  onSuccess: (updated: Reservation) => void;
  onClose: () => void;
}

interface EditSeatsContentProps {
  reservation: Reservation;
  onSuccess: (updated: Reservation) => void;
  onClose: () => void;
}

function sameSeatSelection(a: number[], b: number[]): boolean {
  if (a.length !== b.length) {
    return false;
  }
  const sortedA = [...a].sort((x, y) => x - y);
  const sortedB = [...b].sort((x, y) => x - y);
  return sortedA.every((value, index) => value === sortedB[index]);
}

function EditSeatsContent({ reservation, onSuccess, onClose }: EditSeatsContentProps) {
  const initialSeatIds = useMemo(
    () => reservation.seats.map((seat) => seat.id),
    [reservation.seats],
  );
  const currentSeatIdSet = useMemo(() => new Set(initialSeatIds), [initialSeatIds]);

  const seatsQuery = useShowtimeSeatsQuery(reservation.showtime.id);
  const updateMutation = useUpdateReservationSeatsMutation();

  const [selectedSeatIds, setSelectedSeatIds] = useState<number[]>(initialSeatIds);
  const [limitWarning, setLimitWarning] = useState(false);

  // Treat the user's currently reserved seats as available so they can keep or swap them.
  const mergedSeats = useMemo<Seat[]>(() => {
    const rawSeats = seatsQuery.data?.seats ?? [];
    return rawSeats.map((seat) =>
      currentSeatIdSet.has(seat.id) ? { ...seat, available: true } : seat,
    );
  }, [seatsQuery.data?.seats, currentSeatIdSet]);

  const selectedSeats = useMemo<Seat[]>(() => {
    if (selectedSeatIds.length === 0 || mergedSeats.length === 0) {
      return [];
    }
    const idSet = new Set(selectedSeatIds);
    return mergedSeats.filter((seat) => seat.available && idSet.has(seat.id));
  }, [mergedSeats, selectedSeatIds]);

  const selectedSeatIdSet = useMemo(
    () => new Set(selectedSeats.map((seat) => seat.id)),
    [selectedSeats],
  );

  if (seatsQuery.isPending) {
    return <LoadingState label="Cargando disponibilidad de la sala..." />;
  }

  if (seatsQuery.isError) {
    return (
      <div className="dialog-content-stack">
        <ErrorAlert message="No se pudo cargar el mapa de asientos para esta función." />
        <div className="dialog-actions">
          <Button variant="secondary" autoFocus onClick={onClose}>
            Cerrar
          </Button>
          <Button variant="primary" onClick={() => void seatsQuery.refetch()}>
            Reintentar
          </Button>
        </div>
      </div>
    );
  }

  const { showtime } = seatsQuery.data;
  const hasChanges = !sameSeatSelection(
    selectedSeats.map((seat) => seat.id),
    initialSeatIds,
  );
  const totalCents = selectedSeats.length * showtime.priceCents;
  const seatLabels = selectedSeats.map((seat) => `${seat.row}${seat.number}`);

  function handleToggleSeat(seat: Seat) {
    if (!seat.available) {
      return;
    }

    updateMutation.reset();

    setSelectedSeatIds((current) => {
      const availableSet = new Set(
        mergedSeats.filter((s) => s.available).map((s) => s.id),
      );
      const cleanCurrent = current.filter((id) => availableSet.has(id));

      if (cleanCurrent.includes(seat.id)) {
        setLimitWarning(false);
        return cleanCurrent.filter((id) => id !== seat.id);
      }
      if (cleanCurrent.length >= MAX_SEATS_PER_RESERVATION) {
        setLimitWarning(true);
        return cleanCurrent;
      }
      setLimitWarning(false);
      return [...cleanCurrent, seat.id];
    });
  }

  function handleSave() {
    if (selectedSeats.length === 0 || !hasChanges) {
      return;
    }

    setLimitWarning(false);
    updateMutation.mutate(
      {
        reservationId: reservation.id,
        showtimeId: reservation.showtime.id,
        seatIds: selectedSeats.map((seat) => seat.id),
      },
      {
        onSuccess: (updated) => {
          onSuccess(updated);
        },
        onError: (error) => {
          if (isApiError(error) && (error.status === 409 || error.code === 'SEAT_TAKEN')) {
            void seatsQuery.refetch();
          }
        },
      },
    );
  }

  const errorMessage = updateMutation.isError
    ? getReservationActionErrorMessage(updateMutation.error, 'update')
    : null;

  return (
    <div className="dialog-content-stack">
      <p className="field-hint">
        {reservation.showtime.movieTitle} · {showtime.auditorium} ·{' '}
        {formatShowtimeDateTime(reservation.showtime.startsAt)}
      </p>

      {errorMessage ? <ErrorAlert message={errorMessage} /> : null}

      {limitWarning ? (
        <div className="info-alert" role="status">
          Puedes seleccionar un máximo de {MAX_SEATS_PER_RESERVATION} asientos por reserva.
        </div>
      ) : null}

      <SeatMap
        seats={mergedSeats}
        selectedSeatIds={selectedSeatIdSet}
        disabled={updateMutation.isPending}
        onToggleSeat={handleToggleSeat}
      />

      <dl className="summary-list" aria-live="polite">
        <div className="summary-row">
          <dt>Asientos seleccionados</dt>
          <dd>{seatLabels.length > 0 ? seatLabels.join(', ') : 'Ninguno'}</dd>
        </div>
        <div className="summary-row">
          <dt>Cantidad</dt>
          <dd>
            {selectedSeats.length} de {MAX_SEATS_PER_RESERVATION}
          </dd>
        </div>
        <div className="summary-row summary-total">
          <dt>Nuevo total</dt>
          <dd>{formatPriceCents(totalCents)}</dd>
        </div>
      </dl>

      <div className="dialog-actions">
        <Button
          variant="secondary"
          autoFocus
          disabled={updateMutation.isPending}
          onClick={onClose}
        >
          Cancelar
        </Button>
        <Button
          variant="primary"
          disabled={selectedSeats.length === 0 || !hasChanges || updateMutation.isPending}
          onClick={handleSave}
        >
          {updateMutation.isPending ? 'Guardando cambios...' : 'Guardar cambios'}
        </Button>
      </div>
    </div>
  );
}

export function EditSeatsDialog({ reservation, onSuccess, onClose }: EditSeatsDialogProps) {
  return (
    <ModalDialog
      open={reservation !== null}
      title={reservation ? `Modificar asientos — Reserva #${reservation.id}` : 'Modificar asientos'}
      className="app-dialog-wide"
      onClose={onClose}
    >
      {reservation ? (
        <EditSeatsContent
          key={reservation.id}
          reservation={reservation}
          onSuccess={onSuccess}
          onClose={onClose}
        />
      ) : null}
    </ModalDialog>
  );
}
