import { Button } from '../../../shared/ui/Button';
import { ErrorAlert } from '../../../shared/ui/ErrorAlert';
import { LoadingState } from '../../../shared/ui/LoadingState';
import { usePageTitle } from '../../../shared/ui/usePageTitle';
import { MovieGrid } from '../components/MovieGrid';
import { useMoviesQuery } from '../hooks';

export default function MoviesPage() {
  usePageTitle('Cartelera');
  const moviesQuery = useMoviesQuery();

  return (
    <section className="page-section">
      <div className="page-header">
        <h1 className="page-title">Cartelera</h1>
        <p className="page-lead">
          Explora las películas disponibles y selecciona una función para reservar tus asientos.
        </p>
      </div>

      {moviesQuery.isPending ? <LoadingState label="Cargando cartelera..." /> : null}

      {moviesQuery.isError ? (
        <div className="page-section surface-card">
          <ErrorAlert message="No se pudo cargar la cartelera. Intenta nuevamente." />
          <div>
            <Button variant="primary" onClick={() => void moviesQuery.refetch()}>
              Reintentar
            </Button>
          </div>
        </div>
      ) : null}

      {moviesQuery.isSuccess ? <MovieGrid movies={moviesQuery.data} /> : null}
    </section>
  );
}
