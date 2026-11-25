import { useRef, useState, type FormEvent } from 'react';
import { Button } from '../../../shared/ui/Button';
import { ErrorAlert } from '../../../shared/ui/ErrorAlert';
import type { RegisterInput } from '../types';
import { validateEmail, validateName, validateRegisterPassword } from '../validation';

export interface RegisterFormProps {
  onSubmit: (values: RegisterInput) => void;
  isSubmitting: boolean;
  serverError: string | null;
}

export function RegisterForm({ onSubmit, isSubmitting, serverError }: RegisterFormProps) {
  const [name, setName] = useState('');
  const [email, setEmail] = useState('');
  const [password, setPassword] = useState('');
  const [showPassword, setShowPassword] = useState(false);
  const [touched, setTouched] = useState({ name: false, email: false, password: false });
  const [submitted, setSubmitted] = useState(false);

  const nameRef = useRef<HTMLInputElement>(null);
  const emailRef = useRef<HTMLInputElement>(null);
  const passwordRef = useRef<HTMLInputElement>(null);

  const nameError = touched.name || submitted ? validateName(name) : null;
  const emailError = touched.email || submitted ? validateEmail(email) : null;
  const passwordError = touched.password || submitted ? validateRegisterPassword(password) : null;

  function handleSubmit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setSubmitted(true);

    const nextNameError = validateName(name);
    const nextEmailError = validateEmail(email);
    const nextPasswordError = validateRegisterPassword(password);

    if (nextNameError) {
      nameRef.current?.focus();
      return;
    }
    if (nextEmailError) {
      emailRef.current?.focus();
      return;
    }
    if (nextPasswordError) {
      passwordRef.current?.focus();
      return;
    }

    onSubmit({ name: name.trim(), email: email.trim(), password });
  }

  return (
    <form className="auth-form" onSubmit={handleSubmit} noValidate>
      {serverError ? <ErrorAlert message={serverError} /> : null}

      <div className="form-field">
        <label className="form-label" htmlFor="register-name">
          Nombre completo
        </label>
        <input
          ref={nameRef}
          id="register-name"
          name="name"
          type="text"
          className="form-input"
          autoComplete="name"
          enterKeyHint="next"
          maxLength={100}
          required
          value={name}
          onChange={(event) => setName(event.target.value)}
          onBlur={() => setTouched((prev) => ({ ...prev, name: true }))}
          aria-invalid={nameError ? 'true' : 'false'}
          aria-describedby={nameError ? 'register-name-error' : undefined}
        />
        {nameError ? (
          <p id="register-name-error" className="field-error">
            <span aria-hidden="true">!</span> Error: {nameError}
          </p>
        ) : null}
      </div>

      <div className="form-field">
        <label className="form-label" htmlFor="register-email">
          Correo electrónico
        </label>
        <span id="register-email-hint" className="field-hint">
          Ejemplo: ana@correo.com
        </span>
        <input
          ref={emailRef}
          id="register-email"
          name="email"
          type="email"
          className="form-input"
          autoComplete="username"
          spellCheck={false}
          enterKeyHint="next"
          required
          value={email}
          onChange={(event) => setEmail(event.target.value)}
          onBlur={() => setTouched((prev) => ({ ...prev, email: true }))}
          aria-invalid={emailError ? 'true' : 'false'}
          aria-describedby={
            emailError ? 'register-email-hint register-email-error' : 'register-email-hint'
          }
        />
        {emailError ? (
          <p id="register-email-error" className="field-error">
            <span aria-hidden="true">!</span> Error: {emailError}
          </p>
        ) : null}
      </div>

      <div className="form-field">
        <label className="form-label" htmlFor="register-password">
          Contraseña
        </label>
        <span id="register-password-hint" className="field-hint">
          Entre 8 y 72 caracteres (máximo 72 bytes).
        </span>
        <div className="password-field">
          <input
            ref={passwordRef}
            id="register-password"
            name="password"
            type={showPassword ? 'text' : 'password'}
            className="form-input"
            autoComplete="new-password"
            enterKeyHint="done"
            minLength={8}
            required
            value={password}
            onChange={(event) => setPassword(event.target.value)}
            onBlur={() => setTouched((prev) => ({ ...prev, password: true }))}
            aria-invalid={passwordError ? 'true' : 'false'}
            aria-describedby={
              passwordError
                ? 'register-password-hint register-password-error'
                : 'register-password-hint'
            }
          />
          <button
            type="button"
            className="btn password-toggle"
            aria-pressed={showPassword}
            aria-label={showPassword ? 'Ocultar contraseña' : 'Mostrar contraseña'}
            onClick={() => setShowPassword((prev) => !prev)}
          >
            {showPassword ? 'Ocultar' : 'Mostrar'}
          </button>
        </div>
        {passwordError ? (
          <p id="register-password-error" className="field-error">
            <span aria-hidden="true">!</span> Error: {passwordError}
          </p>
        ) : null}
      </div>

      <div className="form-actions">
        <Button type="submit" variant="primary" disabled={isSubmitting}>
          {isSubmitting ? 'Creando cuenta...' : 'Crear cuenta'}
        </Button>
      </div>
    </form>
  );
}
