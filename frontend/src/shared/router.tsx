import { lazy } from 'react';
import { createBrowserRouter, type RouteObject } from 'react-router-dom';
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
    element: <Layout />,
    children: [
      { index: true, element: <MoviesPage /> },
      { path: 'movies/:movieId', element: <MovieDetailPage /> },
      { path: 'showtimes/:showtimeId', element: <ShowtimeSeatsPage /> },
      { path: 'reservations', element: <ReservationsPage /> },
      { path: 'login', element: <LoginPage /> },
      { path: 'register', element: <RegisterPage /> },
      { path: 'admin', element: <AdminPage /> },
      { path: '*', element: <NotFoundPage /> },
    ],
  },
];

export function createAppRouter() {
  return createBrowserRouter(appRoutes);
}
