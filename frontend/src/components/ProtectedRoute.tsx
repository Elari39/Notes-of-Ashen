import React from 'react';
import { Navigate, Outlet, useLocation } from 'react-router-dom';
import { useAuthStore } from '../store/auth';
import { usePreferenceStore } from '../store/preferences';
import { useShallow } from 'zustand/react/shallow';
import { translate } from '../i18n';
import PagePendingState from './RoutePending';

interface ProtectedRouteProps {
  allowedRoles?: string[];
}

const ProtectedRoute: React.FC<ProtectedRouteProps> = ({ allowedRoles }) => {
  const { user, accessToken, isFetching, isInitialized } = useAuthStore(
    useShallow((state) => ({
      user: state.user,
      accessToken: state.accessToken,
      isFetching: state.isFetching,
      isInitialized: state.isInitialized,
    })),
  );
  const language = usePreferenceStore((state) => state.language);
  const location = useLocation();
  const t = (key: Parameters<typeof translate>[1]) => translate(language, key);

  if (!isInitialized || isFetching) {
    return <PagePendingState label={t('common.loadingAuth')} />;
  }

  if (!accessToken || !user) {
    return <Navigate to="/login" replace state={{ from: location }} />;
  }

  if (allowedRoles && !allowedRoles.includes(user.role)) {
    return <div className="text-center mt-20 text-ochre tracking-widest">{t('protected.forbidden')}</div>;
  }

  return <Outlet />;
};

export default ProtectedRoute;
