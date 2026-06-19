import { useEffect, useState } from 'react';

/**
 * 返回经防抖延迟后的 value 副本。
 * 用于 Markdown 实时预览等"输入即解析"场景，避免每次按键触发全量重解析。
 * 参考 ArticleEditor 草稿保存的 setTimeout 防抖范式。
 */
export function useDebouncedValue<T>(value: T, delayMs = 250): T {
  const [debounced, setDebounced] = useState(value);

  useEffect(() => {
    const timer = window.setTimeout(() => setDebounced(value), delayMs);
    return () => window.clearTimeout(timer);
  }, [value, delayMs]);

  return debounced;
}
