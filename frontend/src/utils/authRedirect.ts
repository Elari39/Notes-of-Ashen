// History state may contain the string written by the session-expiry handler
// or the Location written by ProtectedRoute. Treat both as untrusted input.
export const resolveAuthRedirect = (from: unknown): string => {
  let target: string;
  if (typeof from === 'string') {
    target = from;
  } else if (from && typeof from === 'object' && 'pathname' in from && typeof from.pathname === 'string') {
    const search = 'search' in from ? from.search : '';
    const hash = 'hash' in from ? from.hash : '';
    if (typeof search !== 'string' || typeof hash !== 'string'
      || (search !== '' && !search.startsWith('?')) || (hash !== '' && !hash.startsWith('#'))) return '/';
    target = `${from.pathname}${search}${hash}`;
  } else {
    return '/';
  }
  if (!target.startsWith('/') || target.startsWith('//') || target.includes('\\')
    || [...target].some((char) => char.charCodeAt(0) <= 32 || char.charCodeAt(0) === 127)) return '/';
  try {
    const url = new URL(target, 'https://app.invalid');
    if (url.origin !== 'https://app.invalid' || /^\/login\/?$/i.test(url.pathname)) return '/';
    return `${url.pathname}${url.search}${url.hash}`;
  } catch {
    return '/';
  }
};
