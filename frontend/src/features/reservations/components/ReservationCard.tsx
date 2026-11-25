import { formatPriceCents, formatShowtimeDateTime } from '../../../shared/format';
import { Button } from '../../../shared/ui/Button';
import type { Reservation } from '../../showtimes/types';

export interface ReservationCardProps {
  reservation: Reservation;
  canModify: boolean;
  hasStarted: boolean;
  onEditSeats: (reservation: Reservation) => void;
  onCancelReservation: (reservation: Reservation) => void;
}

export function ReservationCard({
  reservation,
  canModify,
  hasStarted,
  onEditSeats,
  onCancelReservation,
}: ReservationCardProps) {
  const isCancelled = reservation.status === 'cancelled';
  const statusLabel = isCancelled
    ? 'Cancelada'
    : hasStarted
      ? 'Función iniciada'
      : 'Activa';
  const statusClass = isCancelled
    ? 'badge badge-muted'
    : hasStarted
      ? 'badge badge-muted'
      : 'badge badge-active';

  const seatLabels =
    reservation.seats.length > 0
      ? reservation.seats.map((seat) => `${seat.row}${seat.number}`).join(', ')
      : 'Sin asientos';

  return (
    <article
      className={`surface-card reservation-card${isCancelled ? ' is-cancelled' : ''}`}
      aria-labelledby={`reservation-title-${reservation.id}`}
    >
      <div className="reservation-card-header">
        <div>
          <p className="reservation-id">Reserva #{reservation.id}</p>
          <h2 id={`reservation-title-${reservation.id}`} className="reservation-movie-title">
            {reservation.showtime.movieTitle}
          </h2>
        </div>
        <span className={statusClass}>{statusLabel}</span>
      </div>

      <dl className="reservation-meta">
        <div className="reservation-meta-row">
          <dt>Función</dt>
          <dd>{formatShowtimeDateTime(reservation.showtime.startsAt)}</dd>
        </div>
        <div className="reservation-meta-row">
          <dt>Asientos</dt>
          <dd>{seatLabels}</dd>
        </div>
        <div className="reservation-meta-row">
          <dt>Total</dt>
          <dd>{formatPriceCents(reservation.totalCents)}</dd>
        </div>
      </dl>

      {canModify ? (
        <div className="reservation-actions">
          <Button variant="primary" onClick={() => onEditSeats(reservation)}>
            Modificar asientos
          </Button>
          <Button variant="secondary" onClick={() => onCancelReservation(reservation)}>
            Cancelar reserva
          </Button>
        </div>
      ) : hasStarted && !isCancelled ? (
        <p className="field-hint">
          La función ya comenzó; no es posible modificar ni cancelar esta reserva.
        </p>
      ) : null}
    </article>
  );
}
