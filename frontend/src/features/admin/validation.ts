import { isApiError } from '../../shared/api/client';
import { isSafeImageUrl } from '../../shared/format';

export interface MovieFormValues {
  title: string;
  synopsis: string;
  durationMin: string;
  rating: string;
  posterUrl: string;
}

export interface MovieFormErrors {
  title?: string;
  synopsis?: string;
  durationMin?: string;
  rating?: string;
  posterUrl?: string;
}

export interface ShowtimeFormValues {
  auditoriumId: string;
  startsAt: string;
  pricePesos: string;
}

export interface ShowtimeFormErrors {
  auditoriumId?: string;
  startsAt?: string;
  pricePesos?: string;
}

export function validateMovieForm(values: MovieFormValues): MovieFormErrors {
  const errors: MovieFormErrors = {};
  const title = values.title.trim();
  const synopsis = values.synopsis.trim();
  const rating = values.rating.trim();
  const posterUrl = values.posterUrl.trim();
  const duration = Number(values.durationMin);

  if (title.length < 1 || title.length > 200) {
    errors.title = 'El título debe tener entre 1 y 200 caracteres.';
  }
  if (synopsis.length > 2000) {
    errors.synopsis = 'La sinopsis no puede superar 2000 caracteres.';
  }
  if (!Number.isInteger(duration) || duration <= 0) {
    errors.durationMin = 'La duración debe ser un número entero mayor a 0.';
  }
  if (rating.length > 20) {
    errors.rating = 'La clasificación no puede superar 20 caracteres.';
  }
  if (posterUrl.length > 0) {
    if (posterUrl.length > 2048 || !isSafeImageUrl(posterUrl)) {
      errors.posterUrl = 'Ingresa una URL válida que comience con http:// o https://.';
    }
  }

  return errors;
}

export function validateShowtimeForm(
  values: ShowtimeFormValues,
  nowMs: number = Date.now(),
): ShowtimeFormErrors {
  const errors: ShowtimeFormErrors = {};
  const auditoriumId = Number(values.auditoriumId);
  const pricePesos = Number(values.pricePesos);

  if (!Number.isInteger(auditoriumId) || auditoriumId <= 0) {
    errors.auditoriumId = 'Selecciona una sala.';
  }
  if (!values.startsAt.trim()) {
    errors.startsAt = 'Ingresa la fecha y hora de la función.';
  } else {
    const parsed = Date.parse(values.startsAt);
    if (Number.isNaN(parsed)) {
      errors.startsAt = 'La fecha y hora no es válida.';
    } else if (parsed <= nowMs) {
      errors.startsAt = 'La función debe programarse en el futuro.';
    }
  }

  if (values.pricePesos.trim() === '' || !Number.isFinite(pricePesos) || pricePesos < 0) {
    errors.pricePesos = 'Ingresa un precio igual o mayor a 0.';
  }

  return errors;
}

export function toDatetimeLocalValue(isoDate: string): string {
  const date = new Date(isoDate);
  if (Number.isNaN(date.getTime())) {
    return '';
  }
  const year = date.getFullYear();
  const month = String(date.getMonth() + 1).padStart(2, '0');
  const day = String(date.getDate()).padStart(2, '0');
  const hours = String(date.getHours()).padStart(2, '0');
  const minutes = String(date.getMinutes()).padStart(2, '0');
  return `${year}-${month}-${day}T${hours}:${minutes}`;
}

export function formatAdminApiError(error: unknown, fallback: string): string {
  if (!isApiError(error)) {
    return fallback;
  }
  if (error.code === 'HAS_ACTIVE_RESERVATIONS') {
    return 'No se puede completar la acción porque existen reservas activas asociadas.';
  }
  if (error.code === 'FORBIDDEN') {
    return 'No tienes permisos de administrador para realizar esta acción.';
  }
  if (error.code === 'NOT_FOUND') {
    return 'El recurso solicitado ya no existe.';
  }
  return error.message || fallback;
}
