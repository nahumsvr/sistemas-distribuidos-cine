import { Suspense } from 'react';
import { Link, NavLink, Outlet } from 'react-router-dom';
import { useAuth } from '../../features/auth/hooks';
import { LoadingState } from './LoadingState';

export function Layout() {
  const { user, isAuthenticated, isLoading, logout } = useAuth();

  return (
    <>
      <header className="site-header">
        <a href="#main-content" className="skip-link visually-hidden">
          Ir al contenido principal
        </a>
        <div className="header-inner">
          <Link to="/" className="site-brand">
            Cine
          </Link>
          <nav aria-label="Principal">
            <ul className="site-nav-list">
              <li>
                <NavLink to="/" end className="site-nav-link">
                  Cartelera
                </NavLink>
              </li>

              {isAuthenticated && user ? (
                <>
                  <li>
                    <NavLink to="/reservations" className="site-nav-link">
                      Mis reservas
                    </NavLink>
                  </li>
                  {user.role === 'admin' ? (
                    <li>
                      <NavLink to="/admin" className="site-nav-link">
                        Administración
                      </NavLink>
                    </li>
                  ) : null}
                  <li className="site-nav-user">
                    <span className="site-user-name">{user.name}</span>
                    <button type="button" className="btn btn-sm" onClick={logout}>
                      Cerrar sesión
                    </button>
                  </li>
                </>
              ) : !isLoading ? (
                <>
                  <li>
                    <NavLink to="/login" className="site-nav-link">
                      Iniciar sesión
                    </NavLink>
                  </li>
                  <li>
                    <NavLink to="/register" className="site-nav-link">
                      Registrarse
                    </NavLink>
                  </li>
                </>
              ) : null}
            </ul>
          </nav>
        </div>
      </header>

      <main id="main-content" className="site-main" tabIndex={-1}>
        <Suspense fallback={<LoadingState label="Cargando vista..." />}>
          <Outlet />
        </Suspense>
      </main>

      <footer className="site-footer">
        <div className="footer-inner">
          <p>Sistema de reservas de cine</p>
        </div>
      </footer>
    </>
  );
}
