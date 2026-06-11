export const registerServiceWorker = () => {
  if (typeof window === 'undefined' || !import.meta.env.PROD || !('serviceWorker' in navigator)) {
    return;
  }
  window.addEventListener('load', () => {
    navigator.serviceWorker.register('/sw.js').catch(() => undefined);
  });
};
