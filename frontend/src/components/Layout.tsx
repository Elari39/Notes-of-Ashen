import React, { useEffect, useRef, useState } from 'react';
import { Link, useLocation, useNavigate, useOutlet } from 'react-router-dom';
import { AnimatePresence, motion, useReducedMotion } from 'framer-motion';
import { useAuthStore } from '../store/auth';
import { usePreferenceStore } from '../store/preferences';
import { useSiteSettingsStore } from '../store/siteSettings';
import { logout as apiLogout } from '../api/auth';
import { isHttpAvatarUrl, normalizeAvatarUrl } from '../utils/avatar';
import { formatText, translate } from '../i18n';

const Layout: React.FC = () => {
  const { user, accessToken, fetchUser, logout } = useAuthStore();
  const { language, effectiveTheme, setLanguage, setThemePreference } = usePreferenceStore();
  const registrationEnabled = useSiteSettingsStore((state) => state.registrationEnabled);
  const navigate = useNavigate();
  const location = useLocation();
  const outlet = useOutlet();
  const shouldReduceMotion = useReducedMotion();
  const [isPreferenceOpen, setIsPreferenceOpen] = useState(false);
  const preferenceRef = useRef<HTMLDivElement>(null);
  const avatarUrl = normalizeAvatarUrl(user?.avatarUrl);
  const shouldShowAvatar = isHttpAvatarUrl(avatarUrl);
  const t = (key: Parameters<typeof translate>[1]) => translate(language, key);
  const currentLanguageLabel = language === 'zh' ? t('preferences.languageZh') : t('preferences.languageEn');
  const currentThemeLabel = effectiveTheme === 'dark' ? t('preferences.themeDark') : t('preferences.themeLight');
  const pageTransitionKey = location.pathname.startsWith('/admin') ? 'admin' : location.pathname;
  const pageTransition = shouldReduceMotion
    ? { duration: 0.01 }
    : { duration: 0.24, ease: [0.22, 1, 0.36, 1] };
  const pageVariants = shouldReduceMotion
    ? {
        initial: { opacity: 1, y: 0 },
        animate: { opacity: 1, y: 0 },
        exit: { opacity: 1, y: 0 },
      }
    : {
        initial: { opacity: 0, y: 8 },
        animate: { opacity: 1, y: 0 },
        exit: { opacity: 0, y: -4 },
      };

  useEffect(() => {
    if (accessToken && !user) {
      fetchUser();
    }
  }, [accessToken, user, fetchUser]);

  useEffect(() => {
    setIsPreferenceOpen(false);
  }, [location.pathname]);

  useEffect(() => {
    if (!isPreferenceOpen) {
      return undefined;
    }

    const handlePointerDown = (event: MouseEvent) => {
      if (!preferenceRef.current?.contains(event.target as Node)) {
        setIsPreferenceOpen(false);
      }
    };

    const handleKeyDown = (event: KeyboardEvent) => {
      if (event.key === 'Escape') {
        setIsPreferenceOpen(false);
      }
    };

    document.addEventListener('mousedown', handlePointerDown);
    document.addEventListener('keydown', handleKeyDown);

    return () => {
      document.removeEventListener('mousedown', handlePointerDown);
      document.removeEventListener('keydown', handleKeyDown);
    };
  }, [isPreferenceOpen]);

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
          <Link to="/search" className="hover:text-ink transition-colors">
            {t('nav.search')}
          </Link>
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
          ) : registrationEnabled ? (
            <Link to="/login" className="hover:text-ink transition-colors">{t('nav.login')}</Link>
          ) : null}

          <span className="hidden sm:inline opacity-40 select-none">|</span>
          <div ref={preferenceRef} className="relative">
            <button
              type="button"
              aria-haspopup="dialog"
              aria-expanded={isPreferenceOpen}
              aria-label={t('preferences.openLabel')}
              onClick={() => setIsPreferenceOpen((open) => !open)}
              className="group flex items-center gap-2 border border-mountain-grey bg-[var(--paper-soft)] px-3 py-1.5 text-ink transition-colors hover:border-ochre hover:text-ochre"
            >
              <span className="h-2 w-2 rounded-full bg-ochre opacity-70 transition-transform group-hover:scale-125"></span>
              <span>{t('preferences.trigger')}</span>
              <span className="hidden text-[10px] uppercase tracking-[0.18em] text-ink-light opacity-70 md:inline">
                {currentLanguageLabel} / {currentThemeLabel}
              </span>
            </button>

            {isPreferenceOpen && (
              <motion.div
                role="dialog"
                aria-label={t('preferences.title')}
                initial={{ opacity: 0, y: -6 }}
                animate={{ opacity: 1, y: 0 }}
                transition={{ duration: 0.2, ease: 'easeOut' }}
                className="absolute right-0 top-full z-50 mt-4 w-[min(20rem,calc(100vw-3rem))] border border-mountain-grey bg-paper p-4 text-left shadow-[0_24px_80px_rgba(0,0,0,0.16)]"
              >
                <div className="border-b border-mountain-grey border-opacity-60 pb-3">
                  <p className="text-sm font-bold tracking-widest text-ink">{t('preferences.title')}</p>
                  <p className="mt-2 text-xs leading-relaxed tracking-wide text-ink-light opacity-80">
                    {t('preferences.subtitle')}
                  </p>
                  <p className="mt-2 text-[11px] tracking-[0.18em] text-ochre">
                    {formatText(t('preferences.current'), { value: `${currentLanguageLabel} / ${currentThemeLabel}` })}
                  </p>
                </div>

                <div className="mt-4 space-y-4">
                  <div>
                    <p className="mb-2 text-xs tracking-widest text-ink-light">{t('preferences.languageTitle')}</p>
                    <div className="grid grid-cols-2 border border-mountain-grey">
                      <button
                        type="button"
                        onClick={() => setLanguage('zh')}
                        className={`px-3 py-2 text-sm transition-colors ${language === 'zh' ? 'bg-ink text-paper' : 'text-ink-light hover:text-ochre'}`}
                      >
                        {t('preferences.languageZh')}
                      </button>
                      <button
                        type="button"
                        onClick={() => setLanguage('en')}
                        className={`border-l border-mountain-grey px-3 py-2 text-sm transition-colors ${language === 'en' ? 'bg-ink text-paper' : 'text-ink-light hover:text-ochre'}`}
                      >
                        {t('preferences.languageEn')}
                      </button>
                    </div>
                  </div>

                  <div>
                    <p className="mb-2 text-xs tracking-widest text-ink-light">{t('preferences.themeTitle')}</p>
                    <div className="grid grid-cols-2 border border-mountain-grey">
                      <button
                        type="button"
                        aria-label={t('toggle.themeToLight')}
                        onClick={() => setThemePreference('light')}
                        className={`px-3 py-2 text-sm transition-colors ${effectiveTheme === 'light' ? 'bg-ink text-paper' : 'text-ink-light hover:text-ochre'}`}
                      >
                        {t('preferences.themeLight')}
                      </button>
                      <button
                        type="button"
                        aria-label={t('toggle.themeToDark')}
                        onClick={() => setThemePreference('dark')}
                        className={`border-l border-mountain-grey px-3 py-2 text-sm transition-colors ${effectiveTheme === 'dark' ? 'bg-ink text-paper' : 'text-ink-light hover:text-ochre'}`}
                      >
                        {t('preferences.themeDark')}
                      </button>
                    </div>
                  </div>
                </div>
              </motion.div>
            )}
          </div>
        </nav>
      </header>

      <main className="flex-grow flex flex-col w-full px-6 md:px-12 py-12">
        <AnimatePresence mode="wait" initial={false}>
          <motion.div
            key={pageTransitionKey}
            variants={pageVariants}
            initial="initial"
            animate="animate"
            exit="exit"
            transition={pageTransition}
            className="page-transition-shell page-transition-surface flex-grow flex flex-col"
          >
            {outlet}
          </motion.div>
        </AnimatePresence>
      </main>

      <footer className="py-12 text-center text-ink-light opacity-70 text-sm tracking-widest">
        <p>{t('footer.poem')}</p>
        <p className="mt-2">&copy; 2026 {t('brand.nameEn')}. {t('footer.crafted')}</p>
      </footer>
    </div>
  );
};

export default Layout;
