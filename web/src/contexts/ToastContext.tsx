import {
  createContext,
  useCallback,
  useContext,
  useMemo,
  useState,
  type ReactNode,
} from 'react';

type ToastVariant = 'info' | 'success' | 'error' | 'loading';

export type ToastItem = {
  id: string;
  message: string;
  variant: ToastVariant;
};

type ToastContextValue = {
  showToast: (opts: {
    message: string;
    variant?: ToastVariant;
    /** 非 loading 时自动消失毫秒数，默认 3800；loading 不自动消失 */
    duration?: number;
  }) => string;
  dismiss: (id: string) => void;
};

const ToastContext = createContext<ToastContextValue | null>(null);

export function useToast(): ToastContextValue {
  const ctx = useContext(ToastContext);
  if (!ctx) {
    throw new Error('useToast must be used within ToastProvider');
  }
  return ctx;
}

export function ToastProvider({ children }: { children: ReactNode }) {
  const [toasts, setToasts] = useState<ToastItem[]>([]);

  const dismiss = useCallback((id: string) => {
    setToasts((prev) => prev.filter((t) => t.id !== id));
  }, []);

  const showToast = useCallback(
    (opts: { message: string; variant?: ToastVariant; duration?: number }) => {
      const id =
        typeof crypto !== 'undefined' && crypto.randomUUID
          ? crypto.randomUUID()
          : `toast-${Date.now()}-${Math.random()}`;
      const variant = opts.variant ?? 'info';
      setToasts((prev) => [...prev, { id, message: opts.message, variant }]);
      if (variant !== 'loading') {
        const ms = opts.duration ?? 3800;
        window.setTimeout(() => dismiss(id), ms);
      }
      return id;
    },
    [dismiss]
  );

  const value = useMemo(() => ({ showToast, dismiss }), [showToast, dismiss]);

  return (
    <ToastContext.Provider value={value}>
      {children}
      <div className="toast-host" aria-live="polite">
        {toasts.map((t) => (
          <div key={t.id} className={`toast toast-${t.variant}`} role="status">
            {t.variant === 'loading' ? (
              <span className="toast-loading">
                <span className="toast-spinner" aria-hidden />
                {t.message}
              </span>
            ) : (
              t.message
            )}
          </div>
        ))}
      </div>
    </ToastContext.Provider>
  );
}
