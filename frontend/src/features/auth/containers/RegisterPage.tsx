import { Link, Navigate, useLocation, useNavigate } from 'react-router-dom';
import { usePageTitle } from '../../../shared/ui/usePageTitle';
import { getAuthErrorMessage, getSafeRedirectPath } from '../api';
import { RegisterForm } from '../components/RegisterForm';
import { useAuth, useRegisterMutation } from '../hooks';
import type { AuthLocationState, RegisterInput } from '../types';

export default function RegisterPage() {
  usePageTitle('Registrarse');

  const { isAuthenticated } = useAuth();
  const registerMutation = useRegisterMutation();
  const navigate = useNavigate();
  const location = useLocation();

  const locationState = (location.state as AuthLocationState | null) ?? null;
  const redirectTo = getSafeRedirectPath(locationState?.from);

  if (isAuthenticated) {
    return <Navigate to={redirectTo} replace />;
  }

  function handleSubmit(values: RegisterInput) {
    registerMutation.mutate(values, {
      onSuccess: () => {
        navigate(redirectTo, { replace: true });
      },
    });
  }

  const serverError = registerMutation.isError
    ? getAuthErrorMessage(registerMutation.error, 'register')
    : null;

  return (
    <section className="page-section surface-card auth-card">
      <h1 className="page-title">Crear cuenta</h1>
      <p className="page-lead">Regístrate para reservar asientos en línea.</p>

      <RegisterForm
        onSubmit={handleSubmit}
        isSubmitting={registerMutation.isPending}
        serverError={serverError}
      />

      <p className="auth-switch">
        ¿Ya tienes una cuenta?{' '}
        <Link to="/login" state={locationState?.from ? { from: redirectTo } : undefined}>
          Iniciar sesión
        </Link>
      </p>
    </section>
  );
}
