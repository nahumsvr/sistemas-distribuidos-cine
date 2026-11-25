import { useEffect, useId, useRef, type ReactNode } from 'react';

export interface ModalDialogProps {
  open: boolean;
  title: string;
  onClose: () => void;
  children: ReactNode;
  className?: string;
}

export function ModalDialog({
  open,
  title,
  onClose,
  children,
  className = '',
}: ModalDialogProps) {
  const dialogRef = useRef<HTMLDialogElement>(null);
  const titleId = useId();

  useEffect(() => {
    const dialog = dialogRef.current;
    if (!dialog) {
      return;
    }

    if (open) {
      if (!dialog.open) {
        dialog.showModal();
      }
    } else if (dialog.open) {
      dialog.close();
    }
  }, [open]);

  const combinedClass = className ? `app-dialog ${className}` : 'app-dialog';

  return (
    <dialog
      ref={dialogRef}
      className={combinedClass}
      aria-labelledby={titleId}
      onCancel={(event) => {
        event.preventDefault();
        onClose();
      }}
    >
      {open ? (
        <div className="app-dialog-body">
          <div className="app-dialog-header">
            <h1 id={titleId} className="app-dialog-title">
              {title}
            </h1>
          </div>
          {children}
        </div>
      ) : null}
    </dialog>
  );
}
