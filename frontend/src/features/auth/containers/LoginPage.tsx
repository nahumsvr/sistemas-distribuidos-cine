import { usePageTitle } from '../../../shared/ui/usePageTitle';

export default function LoginPage() {
  usePageTitle('Iniciar sesión');

  return (
    <section className="page-section surface-card">
      <h1 className="page-title">Iniciar sesión</h1>
      <p className="page-lead">Accede con tu cuenta para gestionar tus reservas.</p>
    </section>
  );
}
