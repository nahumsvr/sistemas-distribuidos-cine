const EMAIL_PATTERN = /^[^\s@]+@[^\s@]+\.[^\s@]+$/;
const textEncoder = new TextEncoder();

export function utf8ByteLength(value: string): number {
  return textEncoder.encode(value).length;
}

export function validateName(value: string): string | null {
  const trimmed = value.trim();
  const charCount = Array.from(trimmed).length;
  if (charCount < 1) {
    return 'Ingresa tu nombre.';
  }
  if (charCount > 100) {
    return 'El nombre no puede superar los 100 caracteres.';
  }
  return null;
}

export function validateEmail(value: string): string | null {
  const trimmed = value.trim();
  if (!trimmed) {
    return 'Ingresa tu correo electrónico.';
  }
  if (trimmed.length > 254 || !EMAIL_PATTERN.test(trimmed)) {
    return 'Ingresa un correo electrónico válido (por ejemplo, usuario@correo.com).';
  }
  return null;
}

export function validateLoginPassword(value: string): string | null {
  if (!value) {
    return 'Ingresa tu contraseña.';
  }
  return null;
}

export function validateRegisterPassword(value: string): string | null {
  const byteLength = utf8ByteLength(value);
  if (byteLength < 8) {
    return 'La contraseña debe tener al menos 8 caracteres.';
  }
  if (byteLength > 72) {
    return 'La contraseña no puede superar los 72 bytes.';
  }
  return null;
}
