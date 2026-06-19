import { useCallback, useRef, useState } from 'react';
import { AppError, toAppError } from '../utils/error';
import { toast } from '../utils/notify';

/**
 * 写操作通用 hook：统一管理 submitting / error / 成功提示。
 *
 * 设计要点：
 * - 不抢现有 InlineNotice 的位置：error 仍以字符串形式返回，业务侧自行决定展示。
 *   传入 successMessage 时自动 toast.success；showErrorToast 默认关闭，避免与 InlineNotice 双显。
 * - 内部 inflight ref 与 http.ts adapter 层去重互补：ref 拦截"同组件内快速点击"，
 *   adapter 拦截"跨组件并发触发"。两层独立，互不依赖。
 * - 不吞错：onError 收到的是 AppError，便于上层做更细判断（比如读 .code / .status）。
 */
export interface UseSubmitOptions<TArgs extends unknown[], TResult> {
  handler: (...args: TArgs) => Promise<TResult>;
  successMessage?: string;
  errorFallback?: string;
  showErrorToast?: boolean;
  onSuccess?: (result: TResult, ...args: TArgs) => void | Promise<void>;
  onError?: (err: AppError, ...args: TArgs) => void;
}

export interface UseSubmitReturn<TArgs extends unknown[]> {
  submit: (...args: TArgs) => Promise<void>;
  submitting: boolean;
  error: string;
  reset: () => void;
}

export function useSubmit<TArgs extends unknown[] = [], TResult = unknown>(
  options: UseSubmitOptions<TArgs, TResult>,
): UseSubmitReturn<TArgs> {
  const [submitting, setSubmitting] = useState(false);
  const [error, setError] = useState('');
  const inflight = useRef(false);

  // 用 ref 持有最新的 options，避免业务侧每次 render 重建 useCallback 依赖
  const optionsRef = useRef(options);
  optionsRef.current = options;

  const submit = useCallback(async (...args: TArgs) => {
    if (inflight.current) {
      return;
    }
    inflight.current = true;
    setSubmitting(true);
    setError('');
    try {
      const result = await optionsRef.current.handler(...args);
      const { successMessage, onSuccess } = optionsRef.current;
      if (successMessage) {
        toast.success(successMessage);
      }
      if (onSuccess) {
        await onSuccess(result, ...args);
      }
    } catch (e) {
      const { errorFallback, showErrorToast, onError } = optionsRef.current;
      const appErr = toAppError(e, errorFallback);
      setError(appErr.message);
      if (showErrorToast) {
        toast.error(appErr.message);
      }
      if (onError) {
        onError(appErr, ...args);
      }
    } finally {
      inflight.current = false;
      setSubmitting(false);
    }
  }, []);

  const reset = useCallback(() => {
    setError('');
  }, []);

  return { submit, submitting, error, reset };
}
