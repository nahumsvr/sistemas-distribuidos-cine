import { lazy } from 'react';
import { createBrowserRouter, type RouteObject } from 'react-router-dom';
import { AuthProvider } from '../features/auth/AuthProvider';
import { ProtectedRoute } from '../features/auth/ProtectedRoute';
import { Layout } from './ui/Layout';

const MoviesPage = lazy(() => import('../features/movies/containers/MoviesPage'));
const MovieDetailPage = lazy(() => import('../features/movies/containers/MovieDetailPage'));
const ShowtimeSeatsPage = lazy(() => import('../features/showtimes/containers/ShowtimeSeatsPage'));
const ReservationsPage = lazy(() => import('../features/reservations/containers/ReservationsPage'));
const LoginPage = lazy(() => import('../features/auth/containers/LoginPage'));
const RegisterPage = lazy(() => import('../features/auth/containers/RegisterPage'));
const AdminPage = lazy(() => import('../features/admin/containers/AdminPage'));
const NotFoundPage = lazy(() => import('./ui/NotFoundPage'));

export const appRoutes: RouteObject[] = [
  {
    path: '/',
    element: (
      <AuthProvider>
        <Layout />
      </AuthProvider>
    ),
    children: [
      { index: true, element: <MoviesPage /> },
      { path: 'movies/:movieId', element: <MovieDetailPage /> },
      { path: 'showtimes/:showtimeId', element: <ShowtimeSeatsPage /> },
      {
        path: 'reservations',
        element: (
          <ProtectedRoute>
            <ReservationsPage />
          </ProtectedRoute>
        ),
      },
      { path: 'login', element: <LoginPage /> },
      { path: 'register', element: <RegisterPage /> },
      {
        path: 'admin',
        element: (
          <ProtectedRoute requiredRole="admin">
            <AdminPage />
          </ProtectedRoute>
        ),
      },
      { path: '*', element: <NotFoundPage /> },
    ],
  },
];

export function createAppRouter() {
  return createBrowserRouter(appRoutes);
}
