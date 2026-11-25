import { usePageTitle } from '../../../shared/ui/usePageTitle';

export default function RegisterPage() {
  usePageTitle('Registrarse');

  return (
    <section className="page-section surface-card">
      <h1 className="page-title">Crear cuenta</h1>
      <p className="page-lead">Regístrate para reservar asientos en línea.</p>
    </section>
  );
}
