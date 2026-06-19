import { create } from 'zustand';

export type ToastType = 'success' | 'error' | 'info';

export interface ToastItem {
  id: number;
  type: ToastType;
  message: string;
  duration: number;
}

const MAX_TOASTS = 4;
const DEFAULT_DURATION: Record<ToastType, number> = {
  success: 3000,
  info: 3000,
  error: 4000,
};

interface UIState {
  toasts: ToastItem[];
  /** 在途请求数，用于驱动请求级进度条 */
  globalLoading: number;
  pushToast: (toast: Omit<ToastItem, 'id' | 'duration'> & { duration?: number }) => number;
  dismissToast: (id: number) => void;
  incLoading: () => void;
  decLoading: () => void;
}

let toastId = 0;

export const useUIStore = create<UIState>((set) => ({
  toasts: [],
  globalLoading: 0,
  pushToast: ({ type, message, duration }) => {
    toastId += 1;
    const id = toastId;
    const item: ToastItem = { id, type, message, duration: duration ?? DEFAULT_DURATION[type] };
    set((state) => {
      const next = [...state.toasts, item];
      // 超过上限挤掉最早的，保留最新反馈
      return { toasts: next.slice(Math.max(0, next.length - MAX_TOASTS)) };
    });
    return id;
  },
  dismissToast: (id) => {
    set((state) => ({ toasts: state.toasts.filter((toast) => toast.id !== id) }));
  },
  incLoading: () => {
    set((state) => ({ globalLoading: state.globalLoading + 1 }));
  },
  decLoading: () => {
    // 防御性兜底，避免成对错误导致计数泄漏为负
    set((state) => ({ globalLoading: Math.max(0, state.globalLoading - 1) }));
  },
}));
