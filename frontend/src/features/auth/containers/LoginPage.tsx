import { Link, Navigate, useLocation, useNavigate } from 'react-router-dom';
import { usePageTitle } from '../../../shared/ui/usePageTitle';
import { getAuthErrorMessage, getSafeRedirectPath } from '../api';
import { LoginForm } from '../components/LoginForm';
import { useAuth, useLoginMutation } from '../hooks';
import type { AuthLocationState, LoginInput } from '../types';

export default function LoginPage() {
  usePageTitle('Iniciar sesión');

  const { isAuthenticated, sessionExpired } = useAuth();
  const loginMutation = useLoginMutation();
  const navigate = useNavigate();
  const location = useLocation();

  const locationState = (location.state as AuthLocationState | null) ?? null;
  const redirectTo = getSafeRedirectPath(locationState?.from);
  const showSessionExpired = sessionExpired || Boolean(locationState?.sessionExpired);

  if (isAuthenticated) {
    return <Navigate to={redirectTo} replace />;
  }

  function handleSubmit(values: LoginInput) {
    loginMutation.mutate(values, {
      onSuccess: () => {
        navigate(redirectTo, { replace: true });
      },
    });
  }

  const serverError = loginMutation.isError
    ? getAuthErrorMessage(loginMutation.error, 'login')
    : null;

  return (
    <section className="page-section surface-card auth-card">
      <h1 className="page-title">Iniciar sesión</h1>
      <p className="page-lead">Accede con tu cuenta para gestionar tus reservas.</p>

      {showSessionExpired ? (
        <div className="info-alert" role="status">
          Tu sesión expiró. Inicia sesión nuevamente para continuar.
        </div>
      ) : null}

      <LoginForm
        onSubmit={handleSubmit}
        isSubmitting={loginMutation.isPending}
        serverError={serverError}
      />

      <p className="auth-switch">
        ¿Todavía no tienes cuenta?{' '}
        <Link to="/register" state={locationState?.from ? { from: redirectTo } : undefined}>
          Crear cuenta
        </Link>
      </p>
    </section>
  );
}
