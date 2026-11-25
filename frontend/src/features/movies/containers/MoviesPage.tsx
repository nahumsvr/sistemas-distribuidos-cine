import { usePageTitle } from '../../../shared/ui/usePageTitle';

export default function MoviesPage() {
  usePageTitle('Cartelera');

  return (
    <section className="page-section surface-card">
      <h1 className="page-title">Cartelera</h1>
      <p className="page-lead">
        Explora las películas disponibles y selecciona una función para reservar tus asientos.
      </p>
    </section>
  );
}
