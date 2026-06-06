import React from 'react';
import { Navigate, Outlet, useLocation } from 'react-router-dom';
import { useAuthStore } from '../store/auth';
import { usePreferenceStore } from '../store/preferences';
import { translate } from '../i18n';

interface ProtectedRouteProps {
  requireAdmin?: boolean;
  allowedRoles?: string[];
}

const ProtectedRoute: React.FC<ProtectedRouteProps> = ({ requireAdmin = false, allowedRoles }) => {
  const { user, accessToken, isFetching, isInitialized } = useAuthStore();
  const language = usePreferenceStore((state) => state.language);
  const location = useLocation();
  const t = (key: Parameters<typeof translate>[1]) => translate(language, key);

  if (!isInitialized || isFetching) {
    return <div className="flex-grow flex items-center justify-center tracking-widest text-ink-light">{t('common.loadingAuth')}</div>;
  }

  if (!accessToken || !user) {
    return <Navigate to="/login" replace state={{ from: location }} />;
  }

  const roles = allowedRoles || (requireAdmin ? ['admin'] : undefined);

  if (roles && !roles.includes(user.role)) {
    return <div className="text-center mt-20 text-ochre tracking-widest">{t('protected.forbidden')}</div>;
  }

  return <Outlet />;
};

export default ProtectedRoute;
