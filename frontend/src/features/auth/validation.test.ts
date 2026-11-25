import { describe, expect, it } from 'vitest';
import { ApiError } from '../../shared/api/client';
import { getAuthErrorMessage, getSafeRedirectPath } from './api';
import {
  utf8ByteLength,
  validateEmail,
  validateLoginPassword,
  validateName,
  validateRegisterPassword,
} from './validation';

describe('auth validation helpers', () => {
  it('validates names including Unicode characters and max length', () => {
    expect(validateName('')).not.toBeNull();
    expect(validateName('   ')).not.toBeNull();
    expect(validateName('María José Ñandú')).toBeNull();
    expect(validateName('a'.repeat(100))).toBeNull();
    expect(validateName('a'.repeat(101))).not.toBeNull();
  });

  it('validates email format and length', () => {
    expect(validateEmail('')).not.toBeNull();
    expect(validateEmail('not-an-email')).not.toBeNull();
    expect(validateEmail('ana@mail.com')).toBeNull();
  });

  it('validates login and register passwords using UTF-8 byte length', () => {
    expect(validateLoginPassword('')).not.toBeNull();
    expect(validateLoginPassword('x')).toBeNull();

    expect(validateRegisterPassword('1234567')).not.toBeNull();
    expect(validateRegisterPassword('12345678')).toBeNull();
    expect(validateRegisterPassword('a'.repeat(72))).toBeNull();
    expect(validateRegisterPassword('a'.repeat(73))).not.toBeNull();

    // 19 emojis = 76 UTF-8 bytes (> 72 bcrypt limit) even though length in code units is smaller.
    const emojiPassword = '🎬'.repeat(19);
    expect(utf8ByteLength(emojiPassword)).toBe(76);
    expect(validateRegisterPassword(emojiPassword)).not.toBeNull();
  });

  it('allows only safe same-origin relative redirect paths', () => {
    expect(getSafeRedirectPath('/reservations')).toBe('/reservations');
    expect(getSafeRedirectPath('/showtimes/7?step=2')).toBe('/showtimes/7?step=2');
    expect(getSafeRedirectPath('//evil.example/phish')).toBe('/');
    expect(getSafeRedirectPath('https://evil.example')).toBe('/');
    expect(getSafeRedirectPath(undefined)).toBe('/');
  });

  it('maps API errors to clear Spanish messages', () => {
    expect(getAuthErrorMessage(new ApiError(401, 'UNAUTHORIZED', 'bad'), 'login')).toContain(
      'incorrectos',
    );
    expect(getAuthErrorMessage(new ApiError(409, 'EMAIL_TAKEN', 'dup'), 'register')).toContain(
      'Ya existe una cuenta',
    );
    expect(getAuthErrorMessage(new ApiError(429, 'RATE_LIMITED', 'slow', 12), 'login')).toContain(
      '12 segundos',
    );
    expect(getAuthErrorMessage(new Error('network down'), 'login')).toContain('conectar');
  });
});
