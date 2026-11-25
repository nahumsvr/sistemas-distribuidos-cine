export interface LoadingStateProps {
  label?: string;
}

export function LoadingState({ label = 'Cargando...' }: LoadingStateProps) {
  return (
    <div className="loading-state">
      <progress aria-label={label} />
      <span>{label}</span>
    </div>
  );
}
