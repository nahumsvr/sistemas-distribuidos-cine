import type { Movie } from '../types';
import { MovieCard } from './MovieCard';

export interface MovieGridProps {
  movies: Movie[];
}

export function MovieGrid({ movies }: MovieGridProps) {
  if (movies.length === 0) {
    return (
      <div className="surface-card empty-state">
        <p className="page-lead">No hay películas en cartelera por el momento.</p>
      </div>
    );
  }

  return (
    <div className="movies-grid">
      {movies.map((movie) => (
        <MovieCard key={movie.id} movie={movie} />
      ))}
    </div>
  );
}
