import React from 'react';
import { Link, NavLink } from 'react-router-dom';
import type { RouteLoader } from '../routes/lazyRoutes';
import { preloadRoute } from '../routes/lazyRoutes';

type PreloadableProps = {
  preload?: RouteLoader | RouteLoader[];
};

type PreloadLinkProps = React.ComponentProps<typeof Link> & PreloadableProps;
type PreloadNavLinkProps = React.ComponentProps<typeof NavLink> & PreloadableProps;

export const PreloadLink: React.FC<PreloadLinkProps> = ({
  preload,
  onPointerEnter,
  onFocus,
  onPointerDown,
  ...props
}) => {
  const preloadRoutes = () => {
    const loaders = Array.isArray(preload) ? preload : [preload];
    loaders.forEach((loader) => preloadRoute(loader));
  };

  const handlePointerEnter: PreloadLinkProps['onPointerEnter'] = (event) => {
    preloadRoutes();
    onPointerEnter?.(event);
  };
  const handleFocus: PreloadLinkProps['onFocus'] = (event) => {
    preloadRoutes();
    onFocus?.(event);
  };
  const handlePointerDown: PreloadLinkProps['onPointerDown'] = (event) => {
    preloadRoutes();
    onPointerDown?.(event);
  };

  return (
    <Link
      {...props}
      onPointerEnter={handlePointerEnter}
      onFocus={handleFocus}
      onPointerDown={handlePointerDown}
    />
  );
};

export const PreloadNavLink: React.FC<PreloadNavLinkProps> = ({
  preload,
  onPointerEnter,
  onFocus,
  onPointerDown,
  ...props
}) => {
  const preloadRoutes = () => {
    const loaders = Array.isArray(preload) ? preload : [preload];
    loaders.forEach((loader) => preloadRoute(loader));
  };

  const handlePointerEnter: PreloadNavLinkProps['onPointerEnter'] = (event) => {
    preloadRoutes();
    onPointerEnter?.(event);
  };
  const handleFocus: PreloadNavLinkProps['onFocus'] = (event) => {
    preloadRoutes();
    onFocus?.(event);
  };
  const handlePointerDown: PreloadNavLinkProps['onPointerDown'] = (event) => {
    preloadRoutes();
    onPointerDown?.(event);
  };

  return (
    <NavLink
      {...props}
      onPointerEnter={handlePointerEnter}
      onFocus={handleFocus}
      onPointerDown={handlePointerDown}
    />
  );
};
