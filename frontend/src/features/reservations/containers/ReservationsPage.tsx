import { usePageTitle } from '../../../shared/ui/usePageTitle';

export default function ReservationsPage() {
  usePageTitle('Mis reservas');

  return (
    <section className="page-section surface-card">
      <h1 className="page-title">Mis reservas</h1>
      <p className="page-lead">
        Consulta, modifica o cancela tus reservas activas antes de que inicie cada función.
      </p>
    </section>
  );
}
