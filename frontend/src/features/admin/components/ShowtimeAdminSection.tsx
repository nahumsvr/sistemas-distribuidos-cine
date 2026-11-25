import type { Showtime } from '../../movies/types';
import { formatPriceCents, formatShowtimeDateTime } from '../../../shared/format';
import { Button } from '../../../shared/ui/Button';
import { ErrorAlert } from '../../../shared/ui/ErrorAlert';
import { LoadingState } from '../../../shared/ui/LoadingState';

export interface ShowtimeAdminSectionProps {
  movieTitle: string | null;
  showtimes: Showtime[];
  isLoading: boolean;
  errorMessage: string | null;
  onCreateClick: () => void;
  onEditClick: (showtime: Showtime) => void;
  onDeleteClick: (showtime: Showtime) => void;
}

export function ShowtimeAdminSection({
  movieTitle,
  showtimes,
  isLoading,
  errorMessage,
  onCreateClick,
  onEditClick,
  onDeleteClick,
}: ShowtimeAdminSectionProps) {
  return (
    <section className="admin-panel surface-card" aria-labelledby="admin-showtimes-heading">
      <div className="admin-panel-header">
        <h2 id="admin-showtimes-heading" className="section-title">
          {movieTitle ? `Funciones de ${movieTitle}` : 'Funciones'}
        </h2>
        {movieTitle ? (
          <Button
            type="button"
            variant="primary"
            className="btn-sm"
            onClick={onCreateClick}
          >
            Nueva función
          </Button>
        ) : null}
      </div>

      {!movieTitle ? (
        <p className="page-lead">Selecciona una película para gestionar sus funciones.</p>
      ) : isLoading ? (
        <LoadingState label="Cargando funciones..." />
      ) : errorMessage ? (
        <ErrorAlert message={errorMessage} />
      ) : showtimes.length === 0 ? (
        <p className="page-lead">No hay funciones futuras programadas para esta película.</p>
      ) : (
        <ul className="admin-list">
          {showtimes.map((showtime) => {
            const formattedDate = formatShowtimeDateTime(showtime.startsAt);
            return (
              <li key={showtime.id} className="admin-list-item">
                <div className="admin-item-info">
                  <strong>{formattedDate}</strong>
                  <span className="movie-duration">
                    {showtime.auditorium} · {formatPriceCents(showtime.priceCents)}
                  </span>
                </div>
                <div className="inline-actions">
                  <Button
                    type="button"
                    className="btn-sm"
                    aria-label={`Editar función ${formattedDate}`}
                    onClick={() => onEditClick(showtime)}
                  >
                    Editar
                  </Button>
                  <Button
                    type="button"
                    className="btn-sm btn-danger"
                    aria-label={`Eliminar función ${formattedDate}`}
                    onClick={() => onDeleteClick(showtime)}
                  >
                    Eliminar
                  </Button>
                </div>
              </li>
            );
          })}
        </ul>
      )}
    </section>
  );
}
