import { useId, useRef, useState, type FormEvent } from 'react';
import type { Movie } from '../../movies/types';
import { Button } from '../../../shared/ui/Button';
import { ErrorAlert } from '../../../shared/ui/ErrorAlert';
import type { MoviePayload } from '../types';
import {
  validateMovieForm,
  type MovieFormErrors,
  type MovieFormValues,
} from '../validation';

export interface MovieFormProps {
  movie: Movie | null;
  isSubmitting: boolean;
  apiError: string | null;
  onSubmit: (payload: MoviePayload) => void;
  onCancel: () => void;
}

function toInitialValues(movie: Movie | null): MovieFormValues {
  return {
    title: movie?.title ?? '',
    synopsis: movie?.synopsis ?? '',
    durationMin: movie ? String(movie.durationMin) : '',
    rating: movie?.rating ?? '',
    posterUrl: movie?.posterUrl ?? '',
  };
}

export function MovieForm({
  movie,
  isSubmitting,
  apiError,
  onSubmit,
  onCancel,
}: MovieFormProps) {
  const [values, setValues] = useState<MovieFormValues>(() => toInitialValues(movie));
  const [errors, setErrors] = useState<MovieFormErrors>({});
  const [hasSubmitted, setHasSubmitted] = useState(false);

  const headingId = useId();
  const titleId = useId();
  const durationId = useId();
  const ratingId = useId();
  const posterId = useId();
  const synopsisId = useId();

  const titleRef = useRef<HTMLInputElement>(null);
  const durationRef = useRef<HTMLInputElement>(null);
  const posterRef = useRef<HTMLInputElement>(null);
  const synopsisRef = useRef<HTMLTextAreaElement>(null);
  const ratingRef = useRef<HTMLInputElement>(null);

  const updateField = (field: keyof MovieFormValues, value: string) => {
    const next = { ...values, [field]: value };
    setValues(next);
    if (hasSubmitted) {
      setErrors(validateMovieForm(next));
    }
  };

  const handleSubmit = (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    setHasSubmitted(true);
    const nextErrors = validateMovieForm(values);
    setErrors(nextErrors);

    if (Object.keys(nextErrors).length > 0) {
      if (nextErrors.title) titleRef.current?.focus();
      else if (nextErrors.durationMin) durationRef.current?.focus();
      else if (nextErrors.rating) ratingRef.current?.focus();
      else if (nextErrors.posterUrl) posterRef.current?.focus();
      else if (nextErrors.synopsis) synopsisRef.current?.focus();
      return;
    }

    onSubmit({
      title: values.title.trim(),
      synopsis: values.synopsis.trim(),
      durationMin: Number.parseInt(values.durationMin, 10),
      rating: values.rating.trim(),
      posterUrl: values.posterUrl.trim(),
    });
  };

  const isEditing = movie !== null;

  return (
    <form
      className="admin-form surface-card"
      aria-labelledby={headingId}
      onSubmit={handleSubmit}
      noValidate
    >
      <h3 id={headingId} className="section-title">
        {isEditing ? `Editar película: ${movie.title}` : 'Nueva película'}
      </h3>

      {apiError ? <ErrorAlert message={apiError} /> : null}

      <div className="form-field">
        <label className="form-label" htmlFor={titleId}>
          Título <span aria-hidden="true">*</span>
        </label>
        <input
          ref={titleRef}
          id={titleId}
          name="title"
          type="text"
          className="form-input"
          value={values.title}
          onChange={(e) => updateField('title', e.target.value)}
          required
          maxLength={200}
          aria-invalid={errors.title ? 'true' : undefined}
          aria-describedby={errors.title ? `${titleId}-error` : undefined}
        />
        {errors.title ? (
          <p id={`${titleId}-error`} className="field-error" role="status">
            {errors.title}
          </p>
        ) : null}
      </div>

      <div className="admin-form-row">
        <div className="form-field">
          <label className="form-label" htmlFor={durationId}>
            Duración (minutos) <span aria-hidden="true">*</span>
          </label>
          <input
            ref={durationRef}
            id={durationId}
            name="durationMin"
            type="number"
            min={1}
            step={1}
            className="form-input"
            value={values.durationMin}
            onChange={(e) => updateField('durationMin', e.target.value)}
            required
            aria-invalid={errors.durationMin ? 'true' : undefined}
            aria-describedby={errors.durationMin ? `${durationId}-error` : undefined}
          />
          {errors.durationMin ? (
            <p id={`${durationId}-error`} className="field-error" role="status">
              {errors.durationMin}
            </p>
          ) : null}
        </div>

        <div className="form-field">
          <label className="form-label" htmlFor={ratingId}>
            Clasificación
          </label>
          <input
            ref={ratingRef}
            id={ratingId}
            name="rating"
            type="text"
            className="form-input"
            value={values.rating}
            onChange={(e) => updateField('rating', e.target.value)}
            maxLength={20}
            aria-invalid={errors.rating ? 'true' : undefined}
            aria-describedby={errors.rating ? `${ratingId}-error` : undefined}
          />
          {errors.rating ? (
            <p id={`${ratingId}-error`} className="field-error" role="status">
              {errors.rating}
            </p>
          ) : null}
        </div>
      </div>

      <div className="form-field">
        <label className="form-label" htmlFor={posterId}>
          URL del póster
        </label>
        <input
          ref={posterRef}
          id={posterId}
          name="posterUrl"
          type="url"
          className="form-input"
          value={values.posterUrl}
          onChange={(e) => updateField('posterUrl', e.target.value)}
          aria-invalid={errors.posterUrl ? 'true' : undefined}
          aria-describedby={errors.posterUrl ? `${posterId}-error` : `${posterId}-hint`}
        />
        <p id={`${posterId}-hint`} className="field-hint">
          Opcional. Debe comenzar con http:// o https://.
        </p>
        {errors.posterUrl ? (
          <p id={`${posterId}-error`} className="field-error" role="status">
            {errors.posterUrl}
          </p>
        ) : null}
      </div>

      <div className="form-field">
        <label className="form-label" htmlFor={synopsisId}>
          Sinopsis
        </label>
        <textarea
          ref={synopsisRef}
          id={synopsisId}
          name="synopsis"
          rows={3}
          className="form-input form-textarea"
          value={values.synopsis}
          onChange={(e) => updateField('synopsis', e.target.value)}
          maxLength={2000}
          aria-invalid={errors.synopsis ? 'true' : undefined}
          aria-describedby={errors.synopsis ? `${synopsisId}-error` : undefined}
        />
        {errors.synopsis ? (
          <p id={`${synopsisId}-error`} className="field-error" role="status">
            {errors.synopsis}
          </p>
        ) : null}
      </div>

      <div className="inline-actions">
        <Button type="submit" variant="primary" disabled={isSubmitting}>
          {isSubmitting
            ? 'Guardando...'
            : isEditing
              ? 'Actualizar película'
              : 'Guardar película'}
        </Button>
        <Button type="button" onClick={onCancel} disabled={isSubmitting}>
          Cancelar
        </Button>
      </div>
    </form>
  );
}
