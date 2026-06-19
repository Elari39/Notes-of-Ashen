import { useCallback, useEffect, useRef, useState } from 'react';

/**
 * 验证码发送等场景的倒计时 hook。
 * timer 句柄存于 ref，start 前 / reset 时 / 卸载时统一 clearInterval，
 * 避免重复 start 叠加多个 interval 与卸载后的定时器泄漏。
 */
export function useCountdown(initialSeconds = 60) {
  const [remaining, setRemaining] = useState(0);
  const timerRef = useRef<number | null>(null);

  const clearTimer = useCallback(() => {
    if (timerRef.current !== null) {
      window.clearInterval(timerRef.current);
      timerRef.current = null;
    }
  }, []);

  const start = useCallback(
    (seconds: number = initialSeconds) => {
      clearTimer();
      setRemaining(seconds);
      timerRef.current = window.setInterval(() => {
        setRemaining((prev) => {
          if (prev <= 1) {
            clearTimer();
            return 0;
          }
          return prev - 1;
        });
      }, 1000);
    },
    [clearTimer, initialSeconds],
  );

  const reset = useCallback(() => {
    clearTimer();
    setRemaining(0);
  }, [clearTimer]);

  // 卸载时兜底清理，避免组件在倒计时进行中被移除后定时器继续运行。
  useEffect(() => clearTimer, [clearTimer]);

  return {
    remaining,
    isCounting: remaining > 0,
    start,
    reset,
  };
}
