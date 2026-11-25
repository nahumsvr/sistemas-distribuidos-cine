import { useParams } from 'react-router-dom';
import { usePageTitle } from '../../../shared/ui/usePageTitle';

export default function ShowtimeSeatsPage() {
  const { showtimeId } = useParams();
  usePageTitle('Selección de asientos');

  return (
    <section className="page-section surface-card">
      <h1 className="page-title">Selección de asientos</h1>
      <p className="page-lead">Mapa de disponibilidad para la función #{showtimeId}.</p>
    </section>
  );
}
