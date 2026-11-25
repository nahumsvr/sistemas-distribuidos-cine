import { useState } from 'react';
import { Link } from 'react-router-dom';
import { Button } from '../../../shared/ui/Button';
import { ErrorAlert } from '../../../shared/ui/ErrorAlert';
import { LoadingState } from '../../../shared/ui/LoadingState';
import { usePageTitle } from '../../../shared/ui/usePageTitle';
import type { Reservation } from '../../showtimes/types';
import { getReservationActionErrorMessage } from '../api';
import { CancelReservationDialog } from '../components/CancelReservationDialog';
import { EditSeatsDialog } from '../components/EditSeatsDialog';
import { ReservationCard } from '../components/ReservationCard';
import { useCancelReservationMutation, useReservationsQuery } from '../hooks';

export default function ReservationsPage() {
  usePageTitle('Mis reservas');

  const reservationsQuery = useReservationsQuery();
  const cancelMutation = useCancelReservationMutation();

  const [editingReservation, setEditingReservation] = useState<Reservation | null>(null);
  const [cancellingReservation, setCancellingReservation] = useState<Reservation | null>(null);
  const [feedbackMessage, setFeedbackMessage] = useState<string | null>(null);

  if (reservationsQuery.isPending) {
    return <LoadingState label="Cargando tus reservas..." />;
  }

  if (reservationsQuery.isError) {
    return (
      <section className="page-section surface-card">
        <h1 className="page-title">Mis reservas</h1>
        <ErrorAlert message="No se pudieron cargar tus reservas. Intenta nuevamente." />
        <div>
          <Button variant="primary" onClick={() => void reservationsQuery.refetch()}>
            Reintentar
          </Button>
        </div>
      </section>
    );
  }

  const reservations = reservationsQuery.data;
  const nowMs = Date.now();

  function handleEditSuccess(updated: Reservation) {
    const seats = updated.seats.map((seat) => `${seat.row}${seat.number}`).join(', ');
    setEditingReservation(null);
    setFeedbackMessage(`Reserva #${updated.id} actualizada. Asientos: ${seats}.`);
  }

  function handleOpenCancel(reservation: Reservation) {
    cancelMutation.reset();
    setFeedbackMessage(null);
    setCancellingReservation(reservation);
  }

  function handleConfirmCancel() {
    if (!cancellingReservation) {
      return;
    }

    const target = cancellingReservation;
    cancelMutation.mutate(
      {
        reservationId: target.id,
        showtimeId: target.showtime.id,
      },
      {
        onSuccess: () => {
          setCancellingReservation(null);
          setFeedbackMessage(`Reserva #${target.id} cancelada. Los asientos quedaron libres.`);
        },
      },
    );
  }

  const cancelErrorMessage = cancelMutation.isError
    ? getReservationActionErrorMessage(cancelMutation.error, 'cancel')
    : null;

  return (
    <div className="page-section">
      <div className="reservations-header">
        <div>
          <h1 className="page-title">Mis reservas</h1>
          <p className="page-lead">
            Consulta, modifica o cancela tus reservas activas antes de que inicie cada función.
          </p>
        </div>
        <Link to="/" className="btn">
          Explorar cartelera
        </Link>
      </div>

      {feedbackMessage ? (
        <div className="success-alert" role="status">
          {feedbackMessage}
        </div>
      ) : null}

      {reservations.length === 0 ? (
        <section className="surface-card empty-state">
          <h2 className="section-title">Todavía no tienes reservas</h2>
          <p className="page-lead">
            Elige una película y selecciona tus asientos para crear tu primera reserva.
          </p>
          <div>
            <Link to="/" className="btn btn-primary">
              Ver cartelera
            </Link>
          </div>
        </section>
      ) : (
        <div className="reservations-list">
          {reservations.map((reservation) => {
            const startsAtMs = new Date(reservation.showtime.startsAt).getTime();
            const hasStarted = Number.isFinite(startsAtMs) && startsAtMs <= nowMs;
            const canModify = reservation.status === 'active' && !hasStarted;

            return (
              <ReservationCard
                key={reservation.id}
                reservation={reservation}
                canModify={canModify}
                hasStarted={hasStarted}
                onEditSeats={(target) => {
                  setFeedbackMessage(null);
                  setEditingReservation(target);
                }}
                onCancelReservation={handleOpenCancel}
              />
            );
          })}
        </div>
      )}

      <EditSeatsDialog
        reservation={editingReservation}
        onSuccess={handleEditSuccess}
        onClose={() => setEditingReservation(null)}
      />

      <CancelReservationDialog
        reservation={cancellingReservation}
        isSubmitting={cancelMutation.isPending}
        errorMessage={cancelErrorMessage}
        onConfirm={handleConfirmCancel}
        onClose={() => {
          cancelMutation.reset();
          setCancellingReservation(null);
        }}
      />
    </div>
  );
}
