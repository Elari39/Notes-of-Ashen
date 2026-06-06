import React from 'react';
import { Navigate, Outlet } from 'react-router-dom';
import { useAuthStore } from '../store/auth';

interface ProtectedRouteProps {
  requireAdmin?: boolean;
}

const ProtectedRoute: React.FC<ProtectedRouteProps> = ({ requireAdmin = false }) => {
  const { user, accessToken, isFetching } = useAuthStore();

  if (isFetching) {
    return <div className="flex-grow flex items-center justify-center tracking-widest text-ink-light">验证中...</div>;
  }

  if (!accessToken || !user) {
    return <Navigate to="/login" replace />;
  }

  if (requireAdmin && user.role !== 'admin') {
    return <div className="text-center mt-20 text-ochre tracking-widest">非掌卷人，无权入内。</div>;
  }

  return <Outlet />;
};

export default ProtectedRoute;
