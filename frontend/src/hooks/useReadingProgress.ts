import { useEffect, useRef } from 'react';

/**
 * 阅读进度条 hook：返回需挂到进度条元素的 ref。
 * 监听 scroll/resize，通过 requestAnimationFrame 节流，
 * 直接写 ref.style.width，不触发任何 React state 更新，
 * 避免滚动时整页（含 MarkdownRenderer）重渲染。
 */
export function useReadingProgress() {
  const barRef = useRef<HTMLDivElement | null>(null);
  const frameRef = useRef<number | null>(null);

  useEffect(() => {
    const update = () => {
      frameRef.current = null;
      const el = barRef.current;
      if (!el) {
        return;
      }
      const scrollTop = window.scrollY || document.documentElement.scrollTop;
      const maxScroll = Math.max(1, document.documentElement.scrollHeight - window.innerHeight);
      const percent = Math.min(100, Math.max(0, (scrollTop / maxScroll) * 100));
      el.style.width = `${percent}%`;
    };

    const schedule = () => {
      if (frameRef.current !== null) {
        return;
      }
      frameRef.current = window.requestAnimationFrame(update);
    };

    update();
    window.addEventListener('scroll', schedule, { passive: true });
    window.addEventListener('resize', schedule);
    return () => {
      window.removeEventListener('scroll', schedule);
      window.removeEventListener('resize', schedule);
      if (frameRef.current !== null) {
        window.cancelAnimationFrame(frameRef.current);
        frameRef.current = null;
      }
    };
  }, []);

  return barRef;
}
