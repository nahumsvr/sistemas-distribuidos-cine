import { Link } from 'react-router-dom';
import { usePageTitle } from './usePageTitle';

export default function NotFoundPage() {
  usePageTitle('Página no encontrada');

  return (
    <section className="page-section surface-card">
      <h1 className="page-title">Página no encontrada</h1>
      <p className="page-lead">La ruta solicitada no existe en el sistema de reservas.</p>
      <div>
        <Link to="/" className="btn btn-primary">
          Volver a la cartelera
        </Link>
      </div>
    </section>
  );
}
