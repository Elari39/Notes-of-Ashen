export const registerServiceWorker = () => {
  if (typeof window === 'undefined' || !import.meta.env.PROD || !('serviceWorker' in navigator)) {
    return;
  }
  window.addEventListener('load', () => {
    navigator.serviceWorker.register('/sw.js').catch(() => undefined);
  });
};

/**
 * 文章删除/状态变更后通知 Service Worker 清空文章详情缓存，
 * 避免 offline 回退命中陈旧内容。controller 在 SW 首次激活前为 null，
 * 此时无缓存可清，直接跳过。
 */
export const notifyArticleCacheInvalid = () => {
  if (typeof window === 'undefined' || !('serviceWorker' in navigator)) {
    return;
  }
  navigator.serviceWorker?.controller?.postMessage({ type: 'CLEAR_ARTICLE_CACHE' });
};
