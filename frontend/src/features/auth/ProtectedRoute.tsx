import type { ReactNode } from 'react';
import { Link, Navigate, useLocation } from 'react-router-dom';
import { Button } from '../../shared/ui/Button';
import { ErrorAlert } from '../../shared/ui/ErrorAlert';
import { LoadingState } from '../../shared/ui/LoadingState';
import { useAuth } from './hooks';
import type { AuthLocationState, UserRole } from './types';

export interface ProtectedRouteProps {
  children: ReactNode;
  requiredRole?: UserRole;
}

export function ProtectedRoute({ children, requiredRole }: ProtectedRouteProps) {
  const { user, isAuthenticated, isLoading, isLoggingOut, authError, retrySession } = useAuth();
  const location = useLocation();

  if (isLoggingOut) {
    return null;
  }

  if (isLoading) {
    return <LoadingState label="Verificando sesión..." />;
  }

  if (authError) {
    return (
      <section className="page-section surface-card">
        <h1 className="page-title">No se pudo verificar tu sesión</h1>
        <ErrorAlert message="Ocurrió un error al consultar tu cuenta. Intenta nuevamente." />
        <div>
          <Button variant="primary" onClick={retrySession}>
            Reintentar
          </Button>
        </div>
      </section>
    );
  }

  if (!isAuthenticated || !user) {
    const state: AuthLocationState = {
      from: `${location.pathname}${location.search}`,
    };
    return <Navigate to="/login" replace state={state} />;
  }

  if (requiredRole && user.role !== requiredRole) {
    return (
      <section className="page-section surface-card">
        <h1 className="page-title">Acceso restringido</h1>
        <p className="page-lead">
          No tienes permisos suficientes para acceder a esta sección.
        </p>
        <div>
          <Link to="/" className="btn btn-primary">
            Volver a la cartelera
          </Link>
        </div>
      </section>
    );
  }

  return <>{children}</>;
}
