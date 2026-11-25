import { Suspense } from 'react';
import { Link, NavLink, Outlet } from 'react-router-dom';
import { LoadingState } from './LoadingState';

const NAV_ITEMS = [
  { to: '/', label: 'Cartelera', end: true },
  { to: '/reservations', label: 'Mis reservas', end: false },
  { to: '/login', label: 'Iniciar sesión', end: false },
  { to: '/register', label: 'Registrarse', end: false },
] as const;

export function Layout() {
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
              {NAV_ITEMS.map((item) => (
                <li key={item.to}>
                  <NavLink
                    to={item.to}
                    end={item.end}
                    className="site-nav-link"
                  >
                    {item.label}
                  </NavLink>
                </li>
              ))}
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
