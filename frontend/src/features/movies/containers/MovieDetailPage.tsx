import { useParams } from 'react-router-dom';
import { usePageTitle } from '../../../shared/ui/usePageTitle';

export default function MovieDetailPage() {
  const { movieId } = useParams();
  usePageTitle('Detalle de película');

  return (
    <section className="page-section surface-card">
      <h1 className="page-title">Detalle de película</h1>
      <p className="page-lead">Funciones disponibles para la película #{movieId}.</p>
    </section>
  );
}
