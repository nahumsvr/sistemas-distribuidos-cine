import { Link } from 'react-router-dom';
import { formatPriceCents } from '../../../shared/format';
import { Button } from '../../../shared/ui/Button';
import { ErrorAlert } from '../../../shared/ui/ErrorAlert';
import { MAX_SEATS_PER_RESERVATION, type Reservation, type Seat } from '../types';

export interface BookingSummaryProps {
  showtimeId: number;
  priceCents: number;
  selectedSeats: Seat[];
  isAuthenticated: boolean;
  isSubmitting: boolean;
  limitWarning: boolean;
  errorMessage: string | null;
  createdReservation: Reservation | null;
  onConfirm: () => void;
  onClear: () => void;
}

export function BookingSummary({
  showtimeId,
  priceCents,
  selectedSeats,
  isAuthenticated,
  isSubmitting,
  limitWarning,
  errorMessage,
  createdReservation,
  onConfirm,
  onClear,
}: BookingSummaryProps) {
  const totalCents = selectedSeats.length * priceCents;
  const seatLabels = selectedSeats.map((seat) => `${seat.row}${seat.number}`);

  return (
    <aside className="surface-card booking-summary" aria-label="Resumen de selección">
      <h2 className="section-title">Tu selección</h2>

      {createdReservation ? (
        <div className="success-alert" role="status">
          <p className="success-alert-title">
            Reserva #{createdReservation.id} confirmada
          </p>
          <p>
            Asientos:{' '}
            {createdReservation.seats.map((seat) => `${seat.row}${seat.number}`).join(', ')} ·
            Total: {formatPriceCents(createdReservation.totalCents)}
          </p>
          <div className="inline-actions">
            <Link to="/reservations" className="btn btn-primary btn-sm">
              Ver mis reservas
            </Link>
          </div>
        </div>
      ) : null}

      {errorMessage ? <ErrorAlert message={errorMessage} /> : null}

      {limitWarning ? (
        <div className="info-alert" role="status">
          Puedes seleccionar un máximo de {MAX_SEATS_PER_RESERVATION} asientos por reserva.
        </div>
      ) : null}

      <dl className="summary-list" aria-live="polite">
        <div className="summary-row">
          <dt>Asientos</dt>
          <dd>{seatLabels.length > 0 ? seatLabels.join(', ') : 'Ninguno seleccionado'}</dd>
        </div>
        <div className="summary-row">
          <dt>Cantidad</dt>
          <dd>
            {selectedSeats.length} de {MAX_SEATS_PER_RESERVATION}
          </dd>
        </div>
        <div className="summary-row summary-total">
          <dt>Total</dt>
          <dd>{formatPriceCents(totalCents)}</dd>
        </div>
      </dl>

      <div className="summary-actions">
        {isAuthenticated ? (
          <>
            <Button
              variant="primary"
              disabled={selectedSeats.length === 0 || isSubmitting}
              onClick={onConfirm}
            >
              {isSubmitting ? 'Confirmando reserva...' : 'Confirmar reserva'}
            </Button>
            {selectedSeats.length > 0 ? (
              <Button variant="secondary" disabled={isSubmitting} onClick={onClear}>
                Limpiar selección
              </Button>
            ) : null}
          </>
        ) : (
          <>
            <p className="field-hint">
              Inicia sesión para confirmar los asientos seleccionados.
            </p>
            <Link
              to="/login"
              state={{ from: `/showtimes/${showtimeId}` }}
              className="btn btn-primary"
            >
              Inicia sesión para reservar
            </Link>
          </>
        )}
      </div>
    </aside>
  );
}
