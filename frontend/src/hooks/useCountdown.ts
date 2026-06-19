import { useState, useCallback } from 'react';

/**
 * 验证码发送等场景的倒计时 hook。
 * 到 0 自动停止，卸载时由 setInterval 自带的清理回调清空。
 */
export function useCountdown(initialSeconds = 60) {
  const [remaining, setRemaining] = useState(0);

  const start = useCallback(
    (seconds: number = initialSeconds) => {
      setRemaining(seconds);
      const timer = window.setInterval(() => {
        setRemaining((prev) => {
          if (prev <= 1) {
            window.clearInterval(timer);
            return 0;
          }
          return prev - 1;
        });
      }, 1000);
    },
    [initialSeconds],
  );

  const reset = useCallback(() => {
    setRemaining(0);
  }, []);

  return {
    remaining,
    isCounting: remaining > 0,
    start,
    reset,
  };
}
