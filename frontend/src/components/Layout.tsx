import React, { useEffect, useState } from 'react';
import { Outlet, Link, useLocation, useNavigate, useSearchParams } from 'react-router-dom';
import { motion } from 'framer-motion';
import { useAuthStore } from '../store/auth';
import { usePreferenceStore } from '../store/preferences';
import { logout as apiLogout } from '../api/auth';
import { isHttpAvatarUrl, normalizeAvatarUrl } from '../utils/avatar';
import { translate } from '../i18n';

const Layout: React.FC = () => {
  const { user, accessToken, fetchUser, logout } = useAuthStore();
  const { language, effectiveTheme, toggleLanguage, toggleTheme } = usePreferenceStore();
  const navigate = useNavigate();
  const location = useLocation();
  const [searchParams] = useSearchParams();
  const [isSearchOpen, setIsSearchOpen] = useState(false);
  const [searchKeyword, setSearchKeyword] = useState('');
  const avatarUrl = normalizeAvatarUrl(user?.avatarUrl);
  const shouldShowAvatar = isHttpAvatarUrl(avatarUrl);
  const t = (key: Parameters<typeof translate>[1]) => translate(language, key);
  const currentQuery = (searchParams.get('q') || '').trim();

  useEffect(() => {
    if (accessToken && !user) {
      fetchUser();
    }
  }, [accessToken, user, fetchUser]);

  useEffect(() => {
    setSearchKeyword(currentQuery);
    if (currentQuery) {
      setIsSearchOpen(true);
    }
  }, [currentQuery]);

  useEffect(() => {
    if (location.pathname !== '/' && !currentQuery) {
      setSearchKeyword('');
    }
  }, [location.pathname, currentQuery]);

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

  const handleSearchSubmit = (event: React.FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    const nextKeyword = searchKeyword.trim();

    if (!nextKeyword) {
      navigate('/');
      setIsSearchOpen(false);
      return;
    }

    navigate(`/?q=${encodeURIComponent(nextKeyword)}`);
    setIsSearchOpen(true);
  };

  return (
    <div className="min-h-screen flex flex-col font-serif bg-paper text-ink transition-colors duration-300">
      <header className="border-b border-mountain-grey border-opacity-70 py-6 px-6 md:px-12 flex flex-col gap-6 lg:flex-row lg:justify-between lg:items-center">
        <Link to="/" className="text-2xl tracking-widest text-ink hover:text-ochre transition-colors duration-300">
          <span className="font-bold relative">
            {t('brand.name')}
            <span className="absolute -bottom-2 left-0 w-full h-[2px] bg-ochre transform scale-x-0 transition-transform duration-300 origin-left hover:scale-x-100"></span>
          </span>
        </Link>

        <nav className="flex flex-wrap items-center gap-x-6 gap-y-4 text-ink-light tracking-widest text-sm">
          <Link to="/" className="hover:text-ink transition-colors">{t('nav.home')}</Link>
          <Link to="/archive" className="hover:text-ink transition-colors">{t('nav.archive')}</Link>
          <button
            type="button"
            aria-expanded={isSearchOpen}
            onClick={() => setIsSearchOpen((open) => !open)}
            className={`${isSearchOpen ? 'text-ochre' : 'hover:text-ink'} transition-colors`}
          >
            {t('nav.search')}
          </button>
          {user ? (
            <>
              {user.role === 'admin' && (
                <Link to="/admin" className="hover:text-ochre transition-colors font-bold">{t('nav.admin')}</Link>
              )}
              <Link to="/profile" className="flex items-center space-x-2 hover:text-ink transition-colors group">
                {shouldShowAvatar && (
                  <img src={avatarUrl} alt="avatar" className="w-6 h-6 rounded-full border border-mountain-grey group-hover:border-ochre transition-colors object-cover" />
                )}
                <span>{user.nickname || user.account}</span>
              </Link>
              <span className="opacity-50 select-none">|</span>
              <button type="button" className="hover:text-ochre transition-colors" onClick={handleLogout}>
                {t('nav.logout')}
              </button>
            </>
          ) : (
            <Link to="/login" className="hover:text-ink transition-colors">{t('nav.login')}</Link>
          )}

          <span className="hidden sm:inline opacity-40 select-none">|</span>
          <button
            type="button"
            aria-label={t('toggle.languageLabel')}
            onClick={toggleLanguage}
            className="min-w-10 px-3 py-1 border border-mountain-grey text-ink hover:border-ochre hover:text-ochre transition-colors"
          >
            {t('toggle.language')}
          </button>
          <button
            type="button"
            aria-label={effectiveTheme === 'dark' ? t('toggle.themeToLight') : t('toggle.themeToDark')}
            onClick={toggleTheme}
            className="min-w-10 px-3 py-1 border border-mountain-grey text-ink hover:border-ochre hover:text-ochre transition-colors"
          >
            {effectiveTheme === 'dark' ? t('toggle.themeLight') : t('toggle.themeDark')}
          </button>
        </nav>
      </header>

      {isSearchOpen && (
        <div className="border-b border-mountain-grey border-opacity-60 bg-[var(--paper-muted)] px-6 py-5 md:px-12">
          <form onSubmit={handleSearchSubmit} className="mx-auto flex w-full max-w-4xl flex-col gap-3 md:flex-row md:items-center">
            <input
              value={searchKeyword}
              onChange={(event) => setSearchKeyword(event.target.value)}
              aria-label={t('home.searchCurtainPlaceholder')}
              placeholder={t('home.searchCurtainPlaceholder')}
              className="min-w-0 flex-1 border border-mountain-grey bg-transparent px-4 py-3 text-ink outline-none transition-colors placeholder:text-ink-light placeholder:opacity-50 focus:border-ochre"
            />
            <button
              type="submit"
              className="border border-ink px-5 py-3 text-sm tracking-widest text-ink transition-colors hover:bg-ink hover:text-paper"
            >
              {t('home.searchSubmit')}
            </button>
          </form>
        </div>
      )}

      <main className="flex-grow flex flex-col w-full px-6 md:px-12 py-12">
        <motion.div
          initial={{ opacity: 0, y: 10 }}
          animate={{ opacity: 1, y: 0 }}
          exit={{ opacity: 0, y: -10 }}
          transition={{ duration: 0.8, ease: 'easeInOut' }}
          className="flex-grow flex flex-col"
        >
          <Outlet />
        </motion.div>
      </main>

      <footer className="py-12 text-center text-ink-light opacity-70 text-sm tracking-widest">
        <p>{t('footer.poem')}</p>
        <p className="mt-2">&copy; 2026 {t('brand.nameEn')}. {t('footer.crafted')}</p>
      </footer>
    </div>
  );
};

export default Layout;
