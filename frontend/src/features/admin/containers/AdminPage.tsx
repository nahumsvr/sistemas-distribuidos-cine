import { usePageTitle } from '../../../shared/ui/usePageTitle';

export default function AdminPage() {
  usePageTitle('Administración');

  return (
    <section className="page-section surface-card">
      <h1 className="page-title">Administración</h1>
      <p className="page-lead">Gestión de películas y funciones del cine.</p>
    </section>
  );
}
