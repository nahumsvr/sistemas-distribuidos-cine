import { Link } from 'react-router-dom';
import { formatDuration, isSafeImageUrl } from '../../../shared/format';
import type { Movie } from '../types';

export interface MovieCardProps {
  movie: Movie;
}

export function MovieCard({ movie }: MovieCardProps) {
  const hasPoster = isSafeImageUrl(movie.posterUrl);

  return (
    <article className={`surface-card movie-card${hasPoster ? ' has-poster' : ''}`}>
      {hasPoster ? (
        <img
          src={movie.posterUrl}
          alt={`Póster de ${movie.title}`}
          className="movie-poster"
          loading="lazy"
          decoding="async"
        />
      ) : null}

      <header className="movie-card-header">
        <h2 className="movie-card-title">
          <Link to={`/movies/${movie.id}`} className="card-link">
            {movie.title}
          </Link>
        </h2>
        <div className="movie-meta">
          <span className="badge">{movie.rating}</span>
          <span className="movie-duration">{formatDuration(movie.durationMin)}</span>
        </div>
      </header>

      <p className="movie-synopsis">{movie.synopsis}</p>

      <footer className="movie-card-footer">
        <span className="movie-cta" aria-hidden="true">
          Ver funciones →
        </span>
      </footer>
    </article>
  );
}
