import React, { useEffect } from 'react';
import { AnimatePresence, motion, useReducedMotion } from 'framer-motion';
import { useUIStore, type ToastItem, type ToastType } from '../store/ui';
import { usePreferenceStore } from '../store/preferences';
import { translate } from '../i18n';

const toneBorder: Record<ToastType, string> = {
  error: 'border-ochre',
  success: 'border-ink-light',
  info: 'border-mountain-grey',
};

const ToastEntry: React.FC<{ toast: ToastItem }> = ({ toast }) => {
  const dismissToast = useUIStore((state) => state.dismissToast);
  const language = usePreferenceStore((state) => state.language);
  const shouldReduceMotion = useReducedMotion();
  const dismissLabel = translate(language, 'toast.dismiss');

  useEffect(() => {
    const timer = window.setTimeout(() => {
      dismissToast(toast.id);
    }, toast.duration);
    return () => window.clearTimeout(timer);
  }, [toast.id, toast.duration, dismissToast]);

  const motionProps = shouldReduceMotion
    ? { initial: { opacity: 1 }, animate: { opacity: 1 }, exit: { opacity: 1 } }
    : { initial: { opacity: 0, x: 24 }, animate: { opacity: 1, x: 0 }, exit: { opacity: 0, x: 24 } };

  return (
    <motion.div
      layout
      role={toast.type === 'error' ? 'alert' : 'status'}
      aria-live="polite"
      transition={{ duration: 0.22, ease: 'easeOut' }}
      className={`flex max-w-sm items-start gap-3 border-l-2 bg-[var(--notice-bg)] px-4 py-3 text-sm leading-relaxed text-ink shadow-[0_12px_40px_rgba(0,0,0,0.12)] ${toneBorder[toast.type]}`}
      {...motionProps}
    >
      <span className="flex-1">{toast.message}</span>
      <button
        type="button"
        onClick={() => dismissToast(toast.id)}
        aria-label={dismissLabel}
        className="shrink-0 text-ink-light opacity-70 transition-opacity hover:opacity-100"
      >
        ✕
      </button>
    </motion.div>
  );
};

const Toaster: React.FC = () => {
  const toasts = useUIStore((state) => state.toasts);

  return (
    <div className="pointer-events-none fixed right-4 top-4 z-[100] flex w-[calc(100vw-2rem)] max-w-sm flex-col gap-2">
      <AnimatePresence initial={false}>
        {toasts.map((toast) => (
          <div key={toast.id} className="pointer-events-auto">
            <ToastEntry toast={toast} />
          </div>
        ))}
      </AnimatePresence>
    </div>
  );
};

export default Toaster;
