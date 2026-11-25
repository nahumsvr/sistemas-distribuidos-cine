import { useId, useRef, useState, type FormEvent } from 'react';
import type { Showtime } from '../../movies/types';
import { Button } from '../../../shared/ui/Button';
import { ErrorAlert } from '../../../shared/ui/ErrorAlert';
import type { Auditorium, ShowtimePayload } from '../types';
import {
  toDatetimeLocalValue,
  validateShowtimeForm,
  type ShowtimeFormErrors,
  type ShowtimeFormValues,
} from '../validation';

export interface ShowtimeFormProps {
  movieId: number;
  movieTitle: string;
  auditoriums: Auditorium[];
  showtime: Showtime | null;
  isSubmitting: boolean;
  apiError: string | null;
  onSubmit: (payload: ShowtimePayload) => void;
  onCancel: () => void;
}

function resolveInitialAuditoriumId(
  showtime: Showtime | null,
  auditoriums: Auditorium[],
): string {
  if (showtime) {
    const matched = auditoriums.find((a) => a.name === showtime.auditorium);
    if (matched) {
      return String(matched.id);
    }
  }
  return auditoriums.length > 0 ? String(auditoriums[0].id) : '';
}

export function ShowtimeForm({
  movieId,
  movieTitle,
  auditoriums,
  showtime,
  isSubmitting,
  apiError,
  onSubmit,
  onCancel,
}: ShowtimeFormProps) {
  const [values, setValues] = useState<ShowtimeFormValues>(() => ({
    auditoriumId: resolveInitialAuditoriumId(showtime, auditoriums),
    startsAt: showtime ? toDatetimeLocalValue(showtime.startsAt) : '',
    pricePesos: showtime ? (showtime.priceCents / 100).toFixed(2) : '85.00',
  }));
  const [errors, setErrors] = useState<ShowtimeFormErrors>({});
  const [hasSubmitted, setHasSubmitted] = useState(false);

  const headingId = useId();
  const auditoriumInputId = useId();
  const startsAtId = useId();
  const priceId = useId();

  const auditoriumRef = useRef<HTMLSelectElement>(null);
  const startsAtRef = useRef<HTMLInputElement>(null);
  const priceRef = useRef<HTMLInputElement>(null);

  const updateField = (field: keyof ShowtimeFormValues, value: string) => {
    const next = { ...values, [field]: value };
    setValues(next);
    if (hasSubmitted) {
      setErrors(validateShowtimeForm(next));
    }
  };

  const handleSubmit = (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    setHasSubmitted(true);
    const nextErrors = validateShowtimeForm(values);
    setErrors(nextErrors);

    if (Object.keys(nextErrors).length > 0) {
      if (nextErrors.auditoriumId) auditoriumRef.current?.focus();
      else if (nextErrors.startsAt) startsAtRef.current?.focus();
      else if (nextErrors.pricePesos) priceRef.current?.focus();
      return;
    }

    onSubmit({
      movieId,
      auditoriumId: Number.parseInt(values.auditoriumId, 10),
      startsAt: new Date(values.startsAt).toISOString(),
      priceCents: Math.round(Number(values.pricePesos) * 100),
    });
  };

  const isEditing = showtime !== null;

  return (
    <form
      className="admin-form surface-card"
      aria-labelledby={headingId}
      onSubmit={handleSubmit}
      noValidate
    >
      <h3 id={headingId} className="section-title">
        {isEditing ? `Editar función (${movieTitle})` : `Nueva función para ${movieTitle}`}
      </h3>

      {apiError ? <ErrorAlert message={apiError} /> : null}

      <div className="admin-form-row">
        <div className="form-field">
          <label className="form-label" htmlFor={auditoriumInputId}>
            Sala <span aria-hidden="true">*</span>
          </label>
          <select
            ref={auditoriumRef}
            id={auditoriumInputId}
            name="auditoriumId"
            className="form-input"
            value={values.auditoriumId}
            onChange={(e) => updateField('auditoriumId', e.target.value)}
            required
            aria-invalid={errors.auditoriumId ? 'true' : undefined}
            aria-describedby={errors.auditoriumId ? `${auditoriumInputId}-error` : undefined}
          >
            {auditoriums.length === 0 ? (
              <option value="">Sin salas disponibles</option>
            ) : null}
            {auditoriums.map((auditorium) => (
              <option key={auditorium.id} value={auditorium.id}>
                {auditorium.name} ({auditorium.rows * auditorium.cols} asientos)
              </option>
            ))}
          </select>
          {errors.auditoriumId ? (
            <p id={`${auditoriumInputId}-error`} className="field-error" role="status">
              {errors.auditoriumId}
            </p>
          ) : null}
        </div>

        <div className="form-field">
          <label className="form-label" htmlFor={startsAtId}>
            Fecha y hora <span aria-hidden="true">*</span>
          </label>
          <input
            ref={startsAtRef}
            id={startsAtId}
            name="startsAt"
            type="datetime-local"
            className="form-input"
            value={values.startsAt}
            onChange={(e) => updateField('startsAt', e.target.value)}
            required
            aria-invalid={errors.startsAt ? 'true' : undefined}
            aria-describedby={errors.startsAt ? `${startsAtId}-error` : undefined}
          />
          {errors.startsAt ? (
            <p id={`${startsAtId}-error`} className="field-error" role="status">
              {errors.startsAt}
            </p>
          ) : null}
        </div>

        <div className="form-field">
          <label className="form-label" htmlFor={priceId}>
            Precio (MXN) <span aria-hidden="true">*</span>
          </label>
          <input
            ref={priceRef}
            id={priceId}
            name="pricePesos"
            type="number"
            min={0}
            step="0.50"
            className="form-input"
            value={values.pricePesos}
            onChange={(e) => updateField('pricePesos', e.target.value)}
            required
            aria-invalid={errors.pricePesos ? 'true' : undefined}
            aria-describedby={errors.pricePesos ? `${priceId}-error` : undefined}
          />
          {errors.pricePesos ? (
            <p id={`${priceId}-error`} className="field-error" role="status">
              {errors.pricePesos}
            </p>
          ) : null}
        </div>
      </div>

      <div className="inline-actions">
        <Button
          type="submit"
          variant="primary"
          disabled={isSubmitting || auditoriums.length === 0}
        >
          {isSubmitting
            ? 'Guardando...'
            : isEditing
              ? 'Actualizar función'
              : 'Guardar función'}
        </Button>
        <Button type="button" onClick={onCancel} disabled={isSubmitting}>
          Cancelar
        </Button>
      </div>
    </form>
  );
}
