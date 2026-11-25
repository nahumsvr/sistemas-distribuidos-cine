import { useState } from 'react';
import { useMovieShowtimesQuery, useMoviesQuery } from '../../movies/hooks';
import type { Movie, Showtime } from '../../movies/types';
import { formatShowtimeDateTime } from '../../../shared/format';
import { ErrorAlert } from '../../../shared/ui/ErrorAlert';
import { LoadingState } from '../../../shared/ui/LoadingState';
import { usePageTitle } from '../../../shared/ui/usePageTitle';
import { ConfirmDeleteDialog } from '../components/ConfirmDeleteDialog';
import { MovieAdminSection } from '../components/MovieAdminSection';
import { MovieForm } from '../components/MovieForm';
import { ShowtimeAdminSection } from '../components/ShowtimeAdminSection';
import { ShowtimeForm } from '../components/ShowtimeForm';
import {
  useAuditoriumsQuery,
  useCreateMovieMutation,
  useCreateShowtimeMutation,
  useDeleteMovieMutation,
  useDeleteShowtimeMutation,
  useUpdateMovieMutation,
  useUpdateShowtimeMutation,
} from '../hooks';
import type { MoviePayload, ShowtimePayload } from '../types';
import { formatAdminApiError } from '../validation';

type MovieFormMode = { type: 'create' } | { type: 'edit'; movie: Movie } | null;
type ShowtimeFormMode = { type: 'create' } | { type: 'edit'; showtime: Showtime } | null;
type DeleteTarget =
  | { kind: 'movie'; movie: Movie }
  | { kind: 'showtime'; showtime: Showtime }
  | null;

export default function AdminPage() {
  usePageTitle('Administración');

  const moviesQuery = useMoviesQuery();
  const auditoriumsQuery = useAuditoriumsQuery();

  const [selectedMovieId, setSelectedMovieId] = useState<number | null>(null);
  const [movieFormMode, setMovieFormMode] = useState<MovieFormMode>(null);
  const [showtimeFormMode, setShowtimeFormMode] = useState<ShowtimeFormMode>(null);
  const [deleteTarget, setDeleteTarget] = useState<DeleteTarget>(null);
  const [statusMessage, setStatusMessage] = useState<string | null>(null);
  const [actionError, setActionError] = useState<string | null>(null);

  const movies = moviesQuery.data ?? [];
  const activeMovie =
    movies.find((movie) => movie.id === selectedMovieId) ?? movies[0] ?? null;
  const activeMovieId = activeMovie?.id ?? null;

  const showtimesQuery = useMovieShowtimesQuery(activeMovieId);

  const createMovieMutation = useCreateMovieMutation();
  const updateMovieMutation = useUpdateMovieMutation();
  const deleteMovieMutation = useDeleteMovieMutation();
  const createShowtimeMutation = useCreateShowtimeMutation();
  const updateShowtimeMutation = useUpdateShowtimeMutation();
  const deleteShowtimeMutation = useDeleteShowtimeMutation();

  const handleMovieSubmit = (payload: MoviePayload) => {
    setActionError(null);
    setStatusMessage(null);

    if (movieFormMode?.type === 'edit') {
      updateMovieMutation.mutate(
        { movieId: movieFormMode.movie.id, payload },
        {
          onSuccess: (updated) => {
            setMovieFormMode(null);
            setSelectedMovieId(updated.id);
            setStatusMessage(`Película "${updated.title}" actualizada.`);
          },
          onError: (err) => {
            setActionError(formatAdminApiError(err, 'No se pudo actualizar la película.'));
          },
        },
      );
      return;
    }

    createMovieMutation.mutate(payload, {
      onSuccess: (created) => {
        setMovieFormMode(null);
        setSelectedMovieId(created.id);
        setStatusMessage(`Película "${created.title}" creada.`);
      },
      onError: (err) => {
        setActionError(formatAdminApiError(err, 'No se pudo crear la película.'));
      },
    });
  };

  const handleShowtimeSubmit = (payload: ShowtimePayload) => {
    setActionError(null);
    setStatusMessage(null);

    if (showtimeFormMode?.type === 'edit') {
      updateShowtimeMutation.mutate(
        { showtimeId: showtimeFormMode.showtime.id, payload },
        {
          onSuccess: () => {
            setShowtimeFormMode(null);
            setStatusMessage('Función actualizada.');
          },
          onError: (err) => {
            setActionError(formatAdminApiError(err, 'No se pudo actualizar la función.'));
          },
        },
      );
      return;
    }

    createShowtimeMutation.mutate(payload, {
      onSuccess: () => {
        setShowtimeFormMode(null);
        setStatusMessage('Función creada.');
      },
      onError: (err) => {
        setActionError(formatAdminApiError(err, 'No se pudo crear la función.'));
      },
    });
  };

  const handleConfirmDelete = () => {
    if (!deleteTarget) return;
    setActionError(null);
    setStatusMessage(null);

    if (deleteTarget.kind === 'movie') {
      const targetMovie = deleteTarget.movie;
      deleteMovieMutation.mutate(targetMovie.id, {
        onSuccess: () => {
          setDeleteTarget(null);
          if (selectedMovieId === targetMovie.id) {
            setSelectedMovieId(null);
          }
          if (movieFormMode?.type === 'edit' && movieFormMode.movie.id === targetMovie.id) {
            setMovieFormMode(null);
          }
          setStatusMessage(`Película "${targetMovie.title}" eliminada.`);
        },
        onError: (err) => {
          setDeleteTarget(null);
          setActionError(formatAdminApiError(err, 'No se pudo eliminar la película.'));
        },
      });
      return;
    }

    const targetShowtime = deleteTarget.showtime;
    deleteShowtimeMutation.mutate(
      { showtimeId: targetShowtime.id, movieId: targetShowtime.movieId },
      {
        onSuccess: () => {
          setDeleteTarget(null);
          if (
            showtimeFormMode?.type === 'edit' &&
            showtimeFormMode.showtime.id === targetShowtime.id
          ) {
            setShowtimeFormMode(null);
          }
          setStatusMessage('Función eliminada.');
        },
        onError: (err) => {
          setDeleteTarget(null);
          setActionError(formatAdminApiError(err, 'No se pudo eliminar la función.'));
        },
      },
    );
  };

  const isMovieSaving =
    createMovieMutation.isPending || updateMovieMutation.isPending;
  const isShowtimeSaving =
    createShowtimeMutation.isPending || updateShowtimeMutation.isPending;
  const isDeleting =
    deleteMovieMutation.isPending || deleteShowtimeMutation.isPending;

  const movieFormError =
    createMovieMutation.error || updateMovieMutation.error
      ? formatAdminApiError(
          createMovieMutation.error ?? updateMovieMutation.error,
          'No se pudo guardar la película.',
        )
      : null;

  const showtimeFormError =
    createShowtimeMutation.error || updateShowtimeMutation.error
      ? formatAdminApiError(
          createShowtimeMutation.error ?? updateShowtimeMutation.error,
          'No se pudo guardar la función.',
        )
      : null;

  return (
    <section className="page-section">
      <header className="page-header">
        <h1 className="page-title">Administración</h1>
        <p className="page-lead">Gestión de películas y funciones del cine.</p>
      </header>

      {statusMessage ? (
        <div className="success-alert" role="status">
          <p className="success-alert-title">{statusMessage}</p>
        </div>
      ) : null}

      {actionError ? <ErrorAlert message={actionError} /> : null}

      {moviesQuery.isLoading || auditoriumsQuery.isLoading ? (
        <LoadingState label="Cargando panel de administración..." />
      ) : moviesQuery.isError ? (
        <ErrorAlert
          message={formatAdminApiError(
            moviesQuery.error,
            'No se pudieron cargar las películas.',
          )}
        />
      ) : auditoriumsQuery.isError ? (
        <ErrorAlert
          message={formatAdminApiError(
            auditoriumsQuery.error,
            'No se pudieron cargar las salas.',
          )}
        />
      ) : (
        <div className="admin-grid">
          <MovieAdminSection
            movies={movies}
            selectedMovieId={activeMovieId}
            onSelectMovie={(movieId) => {
              setSelectedMovieId(movieId);
              setShowtimeFormMode(null);
            }}
            onCreateClick={() => {
              setActionError(null);
              setMovieFormMode({ type: 'create' });
            }}
            onEditClick={(movie) => {
              setActionError(null);
              setMovieFormMode({ type: 'edit', movie });
            }}
            onDeleteClick={(movie) => {
              setActionError(null);
              setDeleteTarget({ kind: 'movie', movie });
            }}
          />

          {movieFormMode ? (
            <MovieForm
              key={
                movieFormMode.type === 'edit'
                  ? `edit-movie-${movieFormMode.movie.id}`
                  : 'create-movie'
              }
              movie={movieFormMode.type === 'edit' ? movieFormMode.movie : null}
              isSubmitting={isMovieSaving}
              apiError={movieFormError}
              onSubmit={handleMovieSubmit}
              onCancel={() => setMovieFormMode(null)}
            />
          ) : null}

          <ShowtimeAdminSection
            movieTitle={activeMovie?.title ?? null}
            showtimes={showtimesQuery.data ?? []}
            isLoading={showtimesQuery.isLoading}
            errorMessage={
              showtimesQuery.isError
                ? formatAdminApiError(
                    showtimesQuery.error,
                    'No se pudieron cargar las funciones.',
                  )
                : null
            }
            onCreateClick={() => {
              setActionError(null);
              setShowtimeFormMode({ type: 'create' });
            }}
            onEditClick={(showtime) => {
              setActionError(null);
              setShowtimeFormMode({ type: 'edit', showtime });
            }}
            onDeleteClick={(showtime) => {
              setActionError(null);
              setDeleteTarget({ kind: 'showtime', showtime });
            }}
          />

          {showtimeFormMode && activeMovie ? (
            <ShowtimeForm
              key={
                showtimeFormMode.type === 'edit'
                  ? `edit-showtime-${showtimeFormMode.showtime.id}`
                  : `create-showtime-${activeMovie.id}`
              }
              movieId={activeMovie.id}
              movieTitle={activeMovie.title}
              auditoriums={auditoriumsQuery.data ?? []}
              showtime={
                showtimeFormMode.type === 'edit' ? showtimeFormMode.showtime : null
              }
              isSubmitting={isShowtimeSaving}
              apiError={showtimeFormError}
              onSubmit={handleShowtimeSubmit}
              onCancel={() => setShowtimeFormMode(null)}
            />
          ) : null}
        </div>
      )}

      <ConfirmDeleteDialog
        isOpen={deleteTarget !== null}
        title={
          deleteTarget?.kind === 'movie' ? 'Eliminar película' : 'Eliminar función'
        }
        description={
          deleteTarget?.kind === 'movie'
            ? `¿Seguro que deseas eliminar "${deleteTarget.movie.title}"? Esta acción no se puede deshacer.`
            : deleteTarget?.kind === 'showtime'
              ? `¿Seguro que deseas eliminar la función del ${formatShowtimeDateTime(deleteTarget.showtime.startsAt)}? Esta acción no se puede deshacer.`
              : ''
        }
        isDeleting={isDeleting}
        onConfirm={handleConfirmDelete}
        onCancel={() => {
          if (!isDeleting) {
            setDeleteTarget(null);
          }
        }}
      />
    </section>
  );
}
