import { formatShowtimeDateTime } from '../../../shared/format';
import { Button } from '../../../shared/ui/Button';
import { ErrorAlert } from '../../../shared/ui/ErrorAlert';
import type { Reservation } from '../../showtimes/types';
import { ModalDialog } from './ModalDialog';

export interface CancelReservationDialogProps {
  reservation: Reservation | null;
  isSubmitting: boolean;
  errorMessage: string | null;
  onConfirm: () => void;
  onClose: () => void;
}

export function CancelReservationDialog({
  reservation,
  isSubmitting,
  errorMessage,
  onConfirm,
  onClose,
}: CancelReservationDialogProps) {
  const seatLabels = reservation
    ? reservation.seats.map((seat) => `${seat.row}${seat.number}`).join(', ')
    : '';

  return (
    <ModalDialog
      open={reservation !== null}
      title={reservation ? `¿Cancelar reserva #${reservation.id}?` : 'Cancelar reserva'}
      onClose={onClose}
    >
      {reservation ? (
        <div className="dialog-content-stack">
          <p className="page-lead">
            Se liberarán los asientos de <strong>{reservation.showtime.movieTitle}</strong> (
            {formatShowtimeDateTime(reservation.showtime.startsAt)}).
          </p>
          {seatLabels ? (
            <p className="field-hint">Asientos actuales: {seatLabels}</p>
          ) : null}

          {errorMessage ? <ErrorAlert message={errorMessage} /> : null}

          <div className="dialog-actions">
            <Button
              variant="secondary"
              autoFocus
              disabled={isSubmitting}
              onClick={onClose}
            >
              Mantener reserva
            </Button>
            <Button
              variant="primary"
              className="btn-danger"
              disabled={isSubmitting}
              onClick={onConfirm}
            >
              {isSubmitting ? 'Cancelando...' : 'Sí, cancelar reserva'}
            </Button>
          </div>
        </div>
      ) : null}
    </ModalDialog>
  );
}
