import { useEffect } from 'react';

interface ModalProps {
  open: boolean;
  onClose: () => void;
  title: string;
  children: React.ReactNode;
}

export default function Modal({ open, onClose, title, children }: ModalProps) {
  useEffect(() => {
    if (!open) return;
    const onEscape = (e: KeyboardEvent) => {
      if (e.key === 'Escape') onClose();
    };
    document.addEventListener('keydown', onEscape);
    document.body.style.overflow = 'hidden';
    return () => {
      document.removeEventListener('keydown', onEscape);
      document.body.style.overflow = '';
    };
  }, [open, onClose]);

  if (!open) return null;

  return (
    <div className="feeds-modal-overlay" onClick={onClose} role="dialog" aria-modal="true" aria-labelledby="feeds-modal-title">
      <div className="feeds-modal" onClick={(e) => e.stopPropagation()}>
        <div className="feeds-modal-header">
          <h2 id="feeds-modal-title" className="feeds-modal-title">{title}</h2>
          <button type="button" className="feeds-modal-close" onClick={onClose} aria-label="关闭">×</button>
        </div>
        <div className="feeds-modal-body">
          {children}
        </div>
      </div>
    </div>
  );
}
