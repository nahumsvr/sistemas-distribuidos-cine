import { Link, useParams } from 'react-router-dom';
import { isApiError } from '../../../shared/api/client';
import { formatDuration, isSafeImageUrl, parsePositiveId } from '../../../shared/format';
import { Button } from '../../../shared/ui/Button';
import { ErrorAlert } from '../../../shared/ui/ErrorAlert';
import { LoadingState } from '../../../shared/ui/LoadingState';
import { usePageTitle } from '../../../shared/ui/usePageTitle';
import { ShowtimeList } from '../components/ShowtimeList';
import { useMovieQuery, useMovieShowtimesQuery } from '../hooks';

export default function MovieDetailPage() {
  const { movieId: rawMovieId } = useParams();
  const movieId = parsePositiveId(rawMovieId);

  const movieQuery = useMovieQuery(movieId);
  const showtimesQuery = useMovieShowtimesQuery(movieId);

  usePageTitle(movieQuery.data ? movieQuery.data.title : 'Detalle de película');

  if (movieId === null || (movieQuery.isError && isApiError(movieQuery.error) && movieQuery.error.status === 404)) {
    return (
      <section className="page-section surface-card">
        <h1 className="page-title">Película no encontrada</h1>
        <p className="page-lead">La película solicitada no existe o ya no está disponible.</p>
        <div>
          <Link to="/" className="btn btn-primary">
            Volver a la cartelera
          </Link>
        </div>
      </section>
    );
  }

  if (movieQuery.isPending) {
    return <LoadingState label="Cargando información de la película..." />;
  }

  if (movieQuery.isError) {
    return (
      <section className="page-section surface-card">
        <h1 className="page-title">Detalle de película</h1>
        <ErrorAlert message="No se pudo cargar la información de la película." />
        <div className="inline-actions">
          <Button variant="primary" onClick={() => void movieQuery.refetch()}>
            Reintentar
          </Button>
          <Link to="/" className="btn">
            Volver a la cartelera
          </Link>
        </div>
      </section>
    );
  }

  const movie = movieQuery.data;
  const hasPoster = isSafeImageUrl(movie.posterUrl);

  return (
    <div className="page-section">
      <nav aria-label="Ruta de navegación">
        <Link to="/" className="back-link">
          ← Volver a la cartelera
        </Link>
      </nav>

      <article className={`surface-card movie-detail-header${hasPoster ? ' has-poster' : ''}`}>
        {hasPoster ? (
          <img
            src={movie.posterUrl}
            alt={`Póster de ${movie.title}`}
            className="movie-detail-poster"
            decoding="async"
          />
        ) : null}

        <div className="movie-detail-info">
          <h1 className="page-title">{movie.title}</h1>
          <div className="movie-meta">
            <span className="badge">{movie.rating}</span>
            <span className="movie-duration">{formatDuration(movie.durationMin)}</span>
          </div>
          <p className="page-lead">{movie.synopsis}</p>
        </div>
      </article>

      <section className="surface-card page-section" aria-labelledby="showtimes-heading">
        <h2 id="showtimes-heading" className="section-title">
          Funciones disponibles
        </h2>

        {showtimesQuery.isPending ? <LoadingState label="Cargando funciones..." /> : null}

        {showtimesQuery.isError ? (
          <div className="page-section">
            <ErrorAlert message="No se pudieron cargar las funciones disponibles." />
            <div>
              <Button variant="primary" onClick={() => void showtimesQuery.refetch()}>
                Reintentar
              </Button>
            </div>
          </div>
        ) : null}

        {showtimesQuery.isSuccess ? <ShowtimeList showtimes={showtimesQuery.data} /> : null}
      </section>
    </div>
  );
}
