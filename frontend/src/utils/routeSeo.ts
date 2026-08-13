const routeSEOPaths = new Set(['/archive', '/search', '/projects', '/ask']);

export const routeUsesOwnSEO = (pathname: string) => (
  routeSEOPaths.has(pathname.length > 1 ? pathname.replace(/\/+$/, '') : pathname)
  || pathname.startsWith('/article/')
  || pathname.startsWith('/admin/preview/')
);
