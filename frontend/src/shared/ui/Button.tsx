import type { ButtonHTMLAttributes } from 'react';

export interface ButtonProps extends ButtonHTMLAttributes<HTMLButtonElement> {
  variant?: 'primary' | 'secondary';
}

export function Button({
  variant = 'secondary',
  className = '',
  type = 'button',
  ...props
}: ButtonProps) {
  const variantClass = variant === 'primary' ? 'btn btn-primary' : 'btn';
  const combinedClass = className ? `${variantClass} ${className}` : variantClass;
  return <button type={type} className={combinedClass} {...props} />;
}
