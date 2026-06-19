import { usePreferenceStore } from '../store/preferences';
import { useUIStore, type ToastType } from '../store/ui';
import { translate, type TranslationKey } from '../i18n';
import { toAppError } from './error';

/**
 * 通用 toast 通知封装。语言取自 preference store，保证非组件环境（如 http 拦截器）也能拿到当前语言。
 */
const push = (type: ToastType, message: string, duration?: number) => {
  useUIStore.getState().pushToast({ type, message, duration });
};

export const toast = {
  success: (message: string, duration?: number) => push('success', message, duration),
  error: (message: string, duration?: number) => push('error', message, duration),
  info: (message: string, duration?: number) => push('info', message, duration),
};

/**
 * 从任意错误（AppError / Error / 字符串）派生一条 error toast。
 * 消息为空时回退到 fallbackKey 对应的本地化文案。
 */
export function notifyFromError(err: unknown, fallbackKey: TranslationKey): void {
  const language = usePreferenceStore.getState().language;
  const message = toAppError(err).message?.trim() || translate(language, fallbackKey);
  toast.error(message);
}
