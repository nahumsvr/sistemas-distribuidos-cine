import { useRef, useState, type FormEvent } from 'react';
import { Button } from '../../../shared/ui/Button';
import { ErrorAlert } from '../../../shared/ui/ErrorAlert';
import type { LoginInput } from '../types';
import { validateEmail, validateLoginPassword } from '../validation';

export interface LoginFormProps {
  onSubmit: (values: LoginInput) => void;
  isSubmitting: boolean;
  serverError: string | null;
}

export function LoginForm({ onSubmit, isSubmitting, serverError }: LoginFormProps) {
  const [email, setEmail] = useState('');
  const [password, setPassword] = useState('');
  const [showPassword, setShowPassword] = useState(false);
  const [touched, setTouched] = useState({ email: false, password: false });
  const [submitted, setSubmitted] = useState(false);

  const emailRef = useRef<HTMLInputElement>(null);
  const passwordRef = useRef<HTMLInputElement>(null);

  const emailError = touched.email || submitted ? validateEmail(email) : null;
  const passwordError = touched.password || submitted ? validateLoginPassword(password) : null;

  function handleSubmit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setSubmitted(true);

    const nextEmailError = validateEmail(email);
    const nextPasswordError = validateLoginPassword(password);

    if (nextEmailError) {
      emailRef.current?.focus();
      return;
    }
    if (nextPasswordError) {
      passwordRef.current?.focus();
      return;
    }

    onSubmit({ email: email.trim(), password });
  }

  return (
    <form className="auth-form" onSubmit={handleSubmit} noValidate>
      {serverError ? <ErrorAlert message={serverError} /> : null}

      <div className="form-field">
        <label className="form-label" htmlFor="login-email">
          Correo electrónico
        </label>
        <input
          ref={emailRef}
          id="login-email"
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
          aria-describedby={emailError ? 'login-email-error' : undefined}
        />
        {emailError ? (
          <p id="login-email-error" className="field-error">
            <span aria-hidden="true">!</span> Error: {emailError}
          </p>
        ) : null}
      </div>

      <div className="form-field">
        <label className="form-label" htmlFor="login-password">
          Contraseña
        </label>
        <div className="password-field">
          <input
            ref={passwordRef}
            id="login-password"
            name="password"
            type={showPassword ? 'text' : 'password'}
            className="form-input"
            autoComplete="current-password"
            enterKeyHint="done"
            required
            value={password}
            onChange={(event) => setPassword(event.target.value)}
            onBlur={() => setTouched((prev) => ({ ...prev, password: true }))}
            aria-invalid={passwordError ? 'true' : 'false'}
            aria-describedby={passwordError ? 'login-password-error' : undefined}
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
          <p id="login-password-error" className="field-error">
            <span aria-hidden="true">!</span> Error: {passwordError}
          </p>
        ) : null}
      </div>

      <div className="form-actions">
        <Button type="submit" variant="primary" disabled={isSubmitting}>
          {isSubmitting ? 'Iniciando sesión...' : 'Iniciar sesión'}
        </Button>
      </div>
    </form>
  );
}
