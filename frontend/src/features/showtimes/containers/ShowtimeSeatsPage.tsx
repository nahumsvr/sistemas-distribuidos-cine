import { useMemo, useState } from 'react';
import { Link, useParams } from 'react-router-dom';
import { isApiError } from '../../../shared/api/client';
import {
  formatPriceCents,
  formatShowtimeDateTime,
  parsePositiveId,
} from '../../../shared/format';
import { Button } from '../../../shared/ui/Button';
import { ErrorAlert } from '../../../shared/ui/ErrorAlert';
import { LoadingState } from '../../../shared/ui/LoadingState';
import { usePageTitle } from '../../../shared/ui/usePageTitle';
import { useAuth } from '../../auth/hooks';
import { useMovieQuery } from '../../movies/hooks';
import { getReservationErrorMessage } from '../api';
import { BookingSummary } from '../components/BookingSummary';
import { SeatMap } from '../components/SeatMap';
import { useCreateReservationMutation, useShowtimeSeatsQuery } from '../hooks';
import { MAX_SEATS_PER_RESERVATION, type Reservation, type Seat } from '../types';

export default function ShowtimeSeatsPage() {
  const { showtimeId: rawShowtimeId } = useParams();
  const showtimeId = parsePositiveId(rawShowtimeId);
  const { isAuthenticated } = useAuth();

  const seatsQuery = useShowtimeSeatsQuery(showtimeId);
  const showtime = seatsQuery.data?.showtime ?? null;
  const movieQuery = useMovieQuery(showtime ? showtime.movieId : null);

  const createMutation = useCreateReservationMutation(showtimeId ?? 0);

  const [selectedSeatIds, setSelectedSeatIds] = useState<number[]>([]);
  const [limitWarning, setLimitWarning] = useState(false);
  const [createdReservation, setCreatedReservation] = useState<Reservation | null>(null);

  usePageTitle(
    movieQuery.data ? `Asientos — ${movieQuery.data.title}` : 'Selección de asientos',
  );

  // Derive only seats that are still available in the latest query snapshot.
  const selectedSeats = useMemo<Seat[]>(() => {
    const seats = seatsQuery.data?.seats ?? [];
    if (selectedSeatIds.length === 0 || seats.length === 0) {
      return [];
    }
    const idSet = new Set(selectedSeatIds);
    return seats.filter((seat) => seat.available && idSet.has(seat.id));
  }, [seatsQuery.data?.seats, selectedSeatIds]);

  const selectedSeatIdSet = useMemo(
    () => new Set(selectedSeats.map((seat) => seat.id)),
    [selectedSeats],
  );

  if (
    showtimeId === null ||
    (seatsQuery.isError && isApiError(seatsQuery.error) && seatsQuery.error.status === 404)
  ) {
    return (
      <section className="page-section surface-card">
        <h1 className="page-title">Función no encontrada</h1>
        <p className="page-lead">La función solicitada no existe o ya no está disponible.</p>
        <div>
          <Link to="/" className="btn btn-primary">
            Volver a la cartelera
          </Link>
        </div>
      </section>
    );
  }

  if (seatsQuery.isPending) {
    return <LoadingState label="Cargando mapa de asientos..." />;
  }

  if (seatsQuery.isError) {
    return (
      <section className="page-section surface-card">
        <h1 className="page-title">Selección de asientos</h1>
        <ErrorAlert message="No se pudo cargar la disponibilidad de asientos." />
        <div className="inline-actions">
          <Button variant="primary" onClick={() => void seatsQuery.refetch()}>
            Reintentar
          </Button>
          <Link to="/" className="btn">
            Volver a la cartelera
          </Link>
        </div>
      </section>
    );
  }

  const { showtime: activeShowtime, seats } = seatsQuery.data;
  const availableCount = seats.filter((seat) => seat.available).length;

  function handleToggleSeat(seat: Seat) {
    if (!seat.available) {
      return;
    }

    setCreatedReservation(null);
    createMutation.reset();

    setSelectedSeatIds((current) => {
      // Keep only IDs that remain available before checking the 10-seat cap.
      const availableSet = new Set(seats.filter((s) => s.available).map((s) => s.id));
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

  function handleClear() {
    setSelectedSeatIds([]);
    setLimitWarning(false);
    createMutation.reset();
  }

  function handleConfirm() {
    if (selectedSeats.length === 0 || showtimeId === null) {
      return;
    }
    setLimitWarning(false);
    createMutation.mutate(
      {
        showtimeId,
        seatIds: selectedSeats.map((seat) => seat.id),
      },
      {
        onSuccess: (reservation) => {
          setCreatedReservation(reservation);
          setSelectedSeatIds([]);
        },
        onError: (error) => {
          if (isApiError(error) && (error.status === 409 || error.code === 'SEAT_TAKEN')) {
            setSelectedSeatIds([]);
            void seatsQuery.refetch();
          }
        },
      },
    );
  }

  const errorMessage = createMutation.isError
    ? getReservationErrorMessage(createMutation.error)
    : null;

  return (
    <div className="page-section">
      <nav aria-label="Ruta de navegación">
        <Link to={`/movies/${activeShowtime.movieId}`} className="back-link">
          ← {movieQuery.data ? `Volver a ${movieQuery.data.title}` : 'Volver a la película'}
        </Link>
      </nav>

      <header className="surface-card showtime-header">
        <h1 className="page-title">
          {movieQuery.data ? movieQuery.data.title : 'Selección de asientos'}
        </h1>
        <p className="showtime-subtitle">
          <span>{activeShowtime.auditorium}</span>
          <span aria-hidden="true">·</span>
          <span>{formatShowtimeDateTime(activeShowtime.startsAt)}</span>
          <span aria-hidden="true">·</span>
          <span>{formatPriceCents(activeShowtime.priceCents)} por asiento</span>
          <span aria-hidden="true">·</span>
          <span>{availableCount} disponibles</span>
        </p>
      </header>

      <div className="seat-page-layout">
        <section className="surface-card" aria-label="Sala">
          <SeatMap
            seats={seats}
            selectedSeatIds={selectedSeatIdSet}
            onToggleSeat={handleToggleSeat}
            disabled={createMutation.isPending}
          />
        </section>

        <BookingSummary
          showtimeId={activeShowtime.id}
          priceCents={activeShowtime.priceCents}
          selectedSeats={selectedSeats}
          isAuthenticated={isAuthenticated}
          isSubmitting={createMutation.isPending}
          limitWarning={limitWarning}
          errorMessage={errorMessage}
          createdReservation={createdReservation}
          onConfirm={handleConfirm}
          onClear={handleClear}
        />
      </div>
    </div>
  );
}
