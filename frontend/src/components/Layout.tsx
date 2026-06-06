import React, { useEffect } from 'react';
import { Outlet, Link, useNavigate } from 'react-router-dom';
import { motion } from 'framer-motion';
import { useAuthStore } from '../store/auth';
import { logout as apiLogout } from '../api/auth';
import { isHttpAvatarUrl, normalizeAvatarUrl } from '../utils/avatar';

const Layout: React.FC = () => {
  const { user, accessToken, fetchUser, logout } = useAuthStore();
  const navigate = useNavigate();
  const avatarUrl = normalizeAvatarUrl(user?.avatarUrl);
  const shouldShowAvatar = isHttpAvatarUrl(avatarUrl);

  useEffect(() => {
    if (accessToken && !user) {
      fetchUser();
    }
  }, [accessToken, user, fetchUser]);

  const handleLogout = async () => {
    const refreshToken = localStorage.getItem('refreshToken');
    if (refreshToken) {
      try {
        await apiLogout({ refreshToken });
      } catch (e) {
        console.error(e);
      }
    }
    logout();
    navigate('/');
  };

  return (
    <div className="min-h-screen flex flex-col font-serif">
      {/* 极简导航 */}
      <header className="border-b border-mountain-grey border-opacity-50 py-8 px-6 md:px-12 flex justify-between items-center">
        <Link to="/" className="text-2xl tracking-widest text-ink hover:text-ochre transition-colors duration-300">
          <span className="font-bold relative">
            Notes of Ashen
            <span className="absolute -bottom-2 left-0 w-full h-[2px] bg-ochre transform scale-x-0 transition-transform duration-300 origin-left hover:scale-x-100"></span>
          </span>
        </Link>
        <nav className="space-x-6 md:space-x-8 text-ink-light tracking-widest text-sm flex items-center">
          <Link to="/" className="hover:text-ink transition-colors">诗集</Link>
          <Link to="/archive" className="hover:text-ink transition-colors">归韵</Link>
          {user ? (
            <>
              {user.role === 'admin' && (
                <Link to="/admin" className="hover:text-ochre transition-colors font-bold">掌卷</Link>
              )}
              <div className="flex items-center space-x-3">
                <Link to="/profile" className="flex items-center space-x-2 hover:text-ink transition-colors group">
                  {shouldShowAvatar && (
                    <img src={avatarUrl} alt="avatar" className="w-6 h-6 rounded-full border border-mountain-grey group-hover:border-ochre transition-colors object-cover" />
                  )}
                  <span>{user.nickname || user.account}</span>
                </Link>
              </div>
              <span className="opacity-50 select-none">|</span>
              <span className="cursor-pointer hover:text-ochre transition-colors" onClick={handleLogout}>辞别</span>
            </>
          ) : (
            <Link to="/login" className="hover:text-ink transition-colors">结缘</Link>
          )}
        </nav>
      </header>

      {/* 主体内容带渐显动画 */}
      <main className="flex-grow flex flex-col w-full px-6 md:px-12 py-12">
        <motion.div
          initial={{ opacity: 0, y: 10 }}
          animate={{ opacity: 1, y: 0 }}
          exit={{ opacity: 0, y: -10 }}
          transition={{ duration: 0.8, ease: "easeInOut" }}
          className="flex-grow flex flex-col"
        >
          <Outlet />
        </motion.div>
      </main>

      {/* 诗意页脚 */}
      <footer className="py-12 text-center text-ink-light opacity-60 text-sm tracking-widest">
        <p>山高水长，落笔生花。</p>
        <p className="mt-2">&copy; 2026 Notes of Ashen. Crafted with Ink.</p>
      </footer>
    </div>
  );
};

export default Layout;
