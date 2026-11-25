import type { Movie } from '../../movies/types';
import { formatDuration } from '../../../shared/format';
import { Button } from '../../../shared/ui/Button';

export interface MovieAdminSectionProps {
  movies: Movie[];
  selectedMovieId: number | null;
  onSelectMovie: (movieId: number) => void;
  onCreateClick: () => void;
  onEditClick: (movie: Movie) => void;
  onDeleteClick: (movie: Movie) => void;
}

export function MovieAdminSection({
  movies,
  selectedMovieId,
  onSelectMovie,
  onCreateClick,
  onEditClick,
  onDeleteClick,
}: MovieAdminSectionProps) {
  return (
    <section className="admin-panel surface-card" aria-labelledby="admin-movies-heading">
      <div className="admin-panel-header">
        <h2 id="admin-movies-heading" className="section-title">
          Películas
        </h2>
        <Button type="button" variant="primary" className="btn-sm" onClick={onCreateClick}>
          Nueva película
        </Button>
      </div>

      {movies.length === 0 ? (
        <p className="page-lead">No hay películas registradas.</p>
      ) : (
        <ul className="admin-list">
          {movies.map((movie) => {
            const isSelected = movie.id === selectedMovieId;
            return (
              <li
                key={movie.id}
                className={`admin-list-item${isSelected ? ' admin-list-item-active' : ''}`}
              >
                <div className="admin-item-info">
                  <strong>{movie.title}</strong>
                  <span className="movie-duration">
                    {formatDuration(movie.durationMin)}
                    {movie.rating ? ` · ${movie.rating}` : ''}
                  </span>
                </div>
                <div className="inline-actions">
                  <Button
                    type="button"
                    className="btn-sm"
                    aria-pressed={isSelected}
                    onClick={() => onSelectMovie(movie.id)}
                  >
                    Funciones
                  </Button>
                  <Button
                    type="button"
                    className="btn-sm"
                    aria-label={`Editar ${movie.title}`}
                    onClick={() => onEditClick(movie)}
                  >
                    Editar
                  </Button>
                  <Button
                    type="button"
                    className="btn-sm btn-danger"
                    aria-label={`Eliminar ${movie.title}`}
                    onClick={() => onDeleteClick(movie)}
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
