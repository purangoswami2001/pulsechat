import { createContext, useContext, useState, ReactNode } from 'react';
import { AlertCircle, CheckCircle, Info, X } from 'lucide-react';

type AlertType = 'success' | 'error' | 'info';

interface AlertContextType {
  showAlert: (message: string, type?: AlertType) => void;
  showConfirm: (message: string) => Promise<boolean>;
}

const AlertContext = createContext<AlertContextType | undefined>(undefined);

export const useAlert = () => {
  const context = useContext(AlertContext);
  if (!context) {
    throw new Error('useAlert must be used within an AlertProvider');
  }
  return context;
};

interface AlertProviderProps {
  children: ReactNode;
}

interface ToastState {
  id: string;
  message: string;
  type: AlertType;
}

interface ConfirmState {
  message: string;
  resolve: (value: boolean) => void;
}

export const AlertProvider = ({ children }: AlertProviderProps) => {
  const [toasts, setToasts] = useState<ToastState[]>([]);
  const [confirm, setConfirm] = useState<ConfirmState | null>(null);

  const showAlert = (message: string, type: AlertType = 'info') => {
    const id = Math.random().toString(36).substring(2, 9);
    setToasts((prev) => [...prev, { id, message, type }]);

    // Auto dismiss after 4 seconds
    setTimeout(() => {
      setToasts((prev) => prev.filter((t) => t.id !== id));
    }, 4000);
  };

  const showConfirm = (message: string): Promise<boolean> => {
    return new Promise((resolve) => {
      setConfirm({ message, resolve });
    });
  };

  const handleConfirmClose = (value: boolean) => {
    if (confirm) {
      confirm.resolve(value);
      setConfirm(null);
    }
  };

  const removeToast = (id: string) => {
    setToasts((prev) => prev.filter((t) => t.id !== id));
  };

  return (
    <AlertContext.Provider value={{ showAlert, showConfirm }}>
      {children}

      {/* Toast Notification Container */}
      <div className="fixed top-4 right-4 z-[9999] flex flex-col gap-2 w-full max-w-sm">
        {toasts.map((toast) => {
          const isError = toast.type === 'error';
          const isSuccess = toast.type === 'success';

          return (
            <div
              key={toast.id}
              className={`flex items-start gap-3 p-4 rounded-2xl border backdrop-blur-md shadow-lg transition-all duration-300 animate-slide-in-right ${
                isError
                  ? 'bg-rose-500/10 border-rose-500/20 text-rose-600 dark:text-rose-400'
                  : isSuccess
                    ? 'bg-emerald-500/10 border-emerald-500/20 text-emerald-600 dark:text-emerald-400'
                    : 'bg-blue-500/10 border-blue-500/20 text-blue-600 dark:text-blue-400'
              }`}
            >
              {isError && <AlertCircle className="w-5 h-5 shrink-0" />}
              {isSuccess && <CheckCircle className="w-5 h-5 shrink-0" />}
              {!isError && !isSuccess && <Info className="w-5 h-5 shrink-0" />}

              <div className="flex-1 text-sm font-medium leading-relaxed">{toast.message}</div>

              <button
                onClick={() => removeToast(toast.id)}
                className="p-0.5 rounded-lg hover:bg-slate-200/50 dark:hover:bg-slate-800/50 transition-colors shrink-0"
              >
                <X className="w-4 h-4" />
              </button>
            </div>
          );
        })}
      </div>

      {/* Confirmation Modal */}
      {confirm && (
        <div className="fixed inset-0 z-[9999] flex items-center justify-center p-4 bg-slate-900/60 backdrop-blur-sm animate-fade-in">
          <div className="w-full max-w-md glass border border-slate-200/60 dark:border-slate-800/60 rounded-3xl p-6 shadow-2xl animate-scale-in text-center">
            <div className="mx-auto w-12 h-12 rounded-2xl bg-rose-500/10 text-rose-500 flex items-center justify-center mb-4">
              <AlertCircle className="w-6 h-6" />
            </div>

            <h3 className="text-lg font-bold text-slate-800 dark:text-slate-100 mb-2">Confirm Action</h3>
            <p className="text-sm text-slate-500 dark:text-slate-400 mb-6 leading-relaxed">
              {confirm.message}
            </p>

            <div className="flex items-center gap-3">
              <button
                type="button"
                onClick={() => handleConfirmClose(false)}
                className="flex-1 py-3 px-4 rounded-xl border border-slate-200 dark:border-slate-800 text-sm font-semibold text-slate-600 dark:text-slate-300 hover:bg-slate-50 dark:hover:bg-slate-800/40 transition-colors"
              >
                Cancel
              </button>
              <button
                type="button"
                onClick={() => handleConfirmClose(true)}
                className="flex-1 py-3 px-4 rounded-xl bg-rose-600 hover:bg-rose-500 text-white text-sm font-semibold shadow-md shadow-rose-600/10 hover:shadow-rose-600/20 active:scale-[0.98] transition-all"
              >
                Confirm
              </button>
            </div>
          </div>
        </div>
      )}
    </AlertContext.Provider>
  );
};
