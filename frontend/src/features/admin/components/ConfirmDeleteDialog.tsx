import { useEffect, useId, useRef } from 'react';
import { Button } from '../../../shared/ui/Button';

export interface ConfirmDeleteDialogProps {
  isOpen: boolean;
  title: string;
  description: string;
  isDeleting: boolean;
  onConfirm: () => void;
  onCancel: () => void;
}

export function ConfirmDeleteDialog({
  isOpen,
  title,
  description,
  isDeleting,
  onConfirm,
  onCancel,
}: ConfirmDeleteDialogProps) {
  const dialogRef = useRef<HTMLDialogElement>(null);
  const titleId = useId();
  const descId = useId();

  useEffect(() => {
    const dialog = dialogRef.current;
    if (!dialog) return;

    if (isOpen && !dialog.open) {
      dialog.showModal();
    } else if (!isOpen && dialog.open) {
      dialog.close();
    }
  }, [isOpen]);

  return (
    <dialog
      ref={dialogRef}
      className="confirm-dialog"
      aria-labelledby={titleId}
      aria-describedby={descId}
      onCancel={(event) => {
        event.preventDefault();
        if (!isDeleting) {
          onCancel();
        }
      }}
    >
      <div className="confirm-dialog-body">
        <h1 id={titleId} className="section-title">
          {title}
        </h1>
        <p id={descId} className="page-lead">
          {description}
        </p>
        <div className="inline-actions confirm-dialog-actions">
          <Button type="button" onClick={onCancel} disabled={isDeleting} autoFocus>
            Cancelar
          </Button>
          <Button
            type="button"
            className="btn-danger"
            onClick={onConfirm}
            disabled={isDeleting}
          >
            {isDeleting ? 'Eliminando...' : 'Eliminar'}
          </Button>
        </div>
      </div>
    </dialog>
  );
}
