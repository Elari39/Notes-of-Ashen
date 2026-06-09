import React, { useEffect, useRef, useState } from 'react';
import { Link, useLocation, useNavigate, useOutlet } from 'react-router-dom';
import { AnimatePresence, motion, useReducedMotion } from 'framer-motion';
import { useAuthStore } from '../store/auth';
import { usePreferenceStore } from '../store/preferences';
import { useSiteSettingsStore } from '../store/siteSettings';
import { logout as apiLogout } from '../api/auth';
import { isHttpAvatarUrl, normalizeAvatarUrl } from '../utils/avatar';
import { formatText, translate } from '../i18n';
import { useSEO } from '../utils/seo';

const Layout: React.FC = () => {
  const { user, logout } = useAuthStore();
  const { language, effectiveTheme, setLanguage, setThemePreference } = usePreferenceStore();
  const { resumePageEnabled, resumeNavHidden, projectsPageEnabled, projectsNavHidden } = useSiteSettingsStore();
  const navigate = useNavigate();
  const location = useLocation();
  const outlet = useOutlet();
  const shouldReduceMotion = useReducedMotion();
  const [isPreferenceOpen, setIsPreferenceOpen] = useState(false);
  const [isMobileMenuOpen, setIsMobileMenuOpen] = useState(false);
  const headerRef = useRef<HTMLElement>(null);
  const desktopPreferenceRef = useRef<HTMLDivElement>(null);
  const mobilePreferenceRef = useRef<HTMLDivElement>(null);
  const avatarUrl = normalizeAvatarUrl(user?.avatarUrl);
  const shouldShowAvatar = isHttpAvatarUrl(avatarUrl);
  const t = (key: Parameters<typeof translate>[1]) => translate(language, key);
  const currentLanguageLabel = language === 'zh' ? t('preferences.languageZh') : t('preferences.languageEn');
  const currentThemeLabel = effectiveTheme === 'dark' ? t('preferences.themeDark') : t('preferences.themeLight');
  const pageTransitionKey = location.pathname.startsWith('/admin') ? 'admin' : location.pathname;
  const usesRouteSEO = location.pathname.startsWith('/article/') || location.pathname.startsWith('/admin/preview/');
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

  useSEO(undefined, undefined, undefined, !usesRouteSEO);

  useEffect(() => {
    setIsPreferenceOpen(false);
    setIsMobileMenuOpen(false);
  }, [location.pathname]);

  useEffect(() => {
    if (!isPreferenceOpen && !isMobileMenuOpen) {
      return undefined;
    }

    const handlePointerDown = (event: MouseEvent) => {
      const target = event.target as Node;
      const isInsidePreference =
        desktopPreferenceRef.current?.contains(target) || mobilePreferenceRef.current?.contains(target);

      if (isPreferenceOpen && !isInsidePreference) {
        setIsPreferenceOpen(false);
      }

      if (isMobileMenuOpen && !headerRef.current?.contains(target)) {
        setIsMobileMenuOpen(false);
      }
    };

    const handleKeyDown = (event: KeyboardEvent) => {
      if (event.key === 'Escape') {
        setIsPreferenceOpen(false);
        setIsMobileMenuOpen(false);
      }
    };

    document.addEventListener('mousedown', handlePointerDown);
    document.addEventListener('keydown', handleKeyDown);

    return () => {
      document.removeEventListener('mousedown', handlePointerDown);
      document.removeEventListener('keydown', handleKeyDown);
    };
  }, [isPreferenceOpen, isMobileMenuOpen]);

  const handleLogout = async () => {
    const refreshToken = localStorage.getItem('refreshToken');
    if (refreshToken) {
      try {
        await apiLogout({ refreshToken });
      } catch {
        // Local logout should still complete if the server-side token revoke fails.
      }
    }
    logout();
    setIsPreferenceOpen(false);
    setIsMobileMenuOpen(false);
    navigate('/');
  };

  const desktopLinkClass = 'hover:text-ink transition-colors';
  const mobileLinkClass = 'block border-b border-mountain-grey border-opacity-50 px-1 py-3 text-left transition-colors hover:text-ochre';
  const mobileActionClass = 'w-full border-b border-mountain-grey border-opacity-50 px-1 py-3 text-left transition-colors hover:text-ochre';
  const closeMobileMenu = () => setIsMobileMenuOpen(false);

  const renderPreferencePanel = (variant: 'desktop' | 'mobile') => (
    <motion.div
      role="dialog"
      aria-label={t('preferences.title')}
      initial={{ opacity: 0, y: -6 }}
      animate={{ opacity: 1, y: 0 }}
      transition={{ duration: 0.2, ease: 'easeOut' }}
      className={`z-50 border border-mountain-grey bg-paper p-4 text-left shadow-[0_24px_80px_rgba(0,0,0,0.16)] ${
        variant === 'mobile'
          ? 'fixed left-4 right-4 top-24 max-h-[calc(100vh-7rem)] overflow-y-auto md:hidden'
          : 'absolute right-0 top-full mt-4 hidden w-[min(20rem,calc(100vw-3rem))] md:block'
      }`}
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
  );

  const renderPreferenceControl = (ref: React.RefObject<HTMLDivElement>, variant: 'desktop' | 'mobile') => (
    <div ref={ref} className="relative">
      <button
        type="button"
        aria-haspopup="dialog"
        aria-expanded={isPreferenceOpen}
        aria-label={t('preferences.openLabel')}
        onClick={() => setIsPreferenceOpen((open) => !open)}
        className={`group flex items-center gap-2 border border-mountain-grey bg-[var(--paper-soft)] text-ink transition-colors hover:border-ochre hover:text-ochre ${
          variant === 'mobile' ? 'w-full justify-between px-3 py-3' : 'px-3 py-1.5'
        }`}
      >
        <span className="flex items-center gap-2">
          <span className="h-2 w-2 rounded-full bg-ochre opacity-70 transition-transform group-hover:scale-125"></span>
          <span>{t('preferences.trigger')}</span>
        </span>
        <span className={`text-[10px] uppercase tracking-[0.18em] text-ink-light opacity-70 ${variant === 'desktop' ? 'hidden lg:inline' : ''}`}>
          {currentLanguageLabel} / {currentThemeLabel}
        </span>
      </button>

      {isPreferenceOpen && renderPreferencePanel(variant)}
    </div>
  );

  return (
    <div className="min-h-screen flex flex-col font-serif bg-paper text-ink transition-colors duration-300">
      <header ref={headerRef} className="border-b border-mountain-grey border-opacity-70 px-4 py-4 md:px-12 md:py-6">
        <div className="flex items-center justify-between gap-4">
          <Link to="/" className="text-2xl tracking-widest text-ink hover:text-ochre transition-colors duration-300">
            <span className="font-bold relative">
              {t('brand.name')}
              <span className="absolute -bottom-2 left-0 w-full h-[2px] bg-ochre transform scale-x-0 transition-transform duration-300 origin-left hover:scale-x-100"></span>
            </span>
          </Link>

          <button
            type="button"
            aria-expanded={isMobileMenuOpen}
            aria-label={isMobileMenuOpen ? t('nav.menuClose') : t('nav.menuOpen')}
            onClick={() => {
              setIsMobileMenuOpen((open) => !open);
              setIsPreferenceOpen(false);
            }}
            className="flex h-10 w-10 shrink-0 flex-col items-center justify-center gap-1.5 border border-mountain-grey text-ink transition-colors hover:border-ochre hover:text-ochre md:hidden"
          >
            <span className={`h-px w-5 bg-current transition-transform ${isMobileMenuOpen ? 'translate-y-2 rotate-45' : ''}`}></span>
            <span className={`h-px w-5 bg-current transition-opacity ${isMobileMenuOpen ? 'opacity-0' : 'opacity-100'}`}></span>
            <span className={`h-px w-5 bg-current transition-transform ${isMobileMenuOpen ? '-translate-y-2 -rotate-45' : ''}`}></span>
          </button>

          <nav className="hidden flex-wrap items-center justify-end gap-x-6 gap-y-4 text-sm tracking-widest text-ink-light md:flex">
            <Link to="/" className={desktopLinkClass}>{t('nav.home')}</Link>
            <Link to="/archive" className={desktopLinkClass}>{t('nav.archive')}</Link>
            <Link to="/search" className={desktopLinkClass}>{t('nav.search')}</Link>
            {projectsPageEnabled && !projectsNavHidden && (
              <Link to="/projects" className={desktopLinkClass}>{language === 'zh' ? '项目' : 'Projects'}</Link>
            )}
            {resumePageEnabled && !resumeNavHidden && (
              <Link to="/resume" className={desktopLinkClass}>{language === 'zh' ? '简介' : 'About'}</Link>
            )}
            {user ? (
              <>
                {(user.role === 'admin' || user.role === 'editor') && (
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
              <Link to="/login" className={desktopLinkClass}>{t('nav.login')}</Link>
            )}

            <span className="opacity-40 select-none">|</span>
            {renderPreferenceControl(desktopPreferenceRef, 'desktop')}
          </nav>
        </div>

        {isMobileMenuOpen && (
          <motion.nav
            initial={{ opacity: 0, y: -6 }}
            animate={{ opacity: 1, y: 0 }}
            transition={{ duration: 0.18, ease: 'easeOut' }}
            className="mt-4 border-t border-mountain-grey border-opacity-70 pt-2 text-sm tracking-widest text-ink-light md:hidden"
          >
            <Link to="/" onClick={closeMobileMenu} className={mobileLinkClass}>{t('nav.home')}</Link>
            <Link to="/archive" onClick={closeMobileMenu} className={mobileLinkClass}>{t('nav.archive')}</Link>
            <Link to="/search" onClick={closeMobileMenu} className={mobileLinkClass}>{t('nav.search')}</Link>
            {projectsPageEnabled && !projectsNavHidden && (
              <Link to="/projects" onClick={closeMobileMenu} className={mobileLinkClass}>{language === 'zh' ? '项目' : 'Projects'}</Link>
            )}
            {resumePageEnabled && !resumeNavHidden && (
              <Link to="/resume" onClick={closeMobileMenu} className={mobileLinkClass}>{language === 'zh' ? '简介' : 'About'}</Link>
            )}
            {user ? (
              <>
                {(user.role === 'admin' || user.role === 'editor') && (
                  <Link to="/admin" onClick={closeMobileMenu} className={`${mobileLinkClass} font-bold text-ochre`}>
                    {t('nav.admin')}
                  </Link>
                )}
                <Link to="/profile" onClick={closeMobileMenu} className={`${mobileLinkClass} flex items-center gap-3`}>
                  {shouldShowAvatar && (
                    <img src={avatarUrl} alt="avatar" className="h-7 w-7 rounded-full border border-mountain-grey object-cover" />
                  )}
                  <span>{user.nickname || user.account}</span>
                </Link>
                <button type="button" className={mobileActionClass} onClick={handleLogout}>
                  {t('nav.logout')}
                </button>
              </>
            ) : (
              <Link to="/login" onClick={closeMobileMenu} className={mobileLinkClass}>{t('nav.login')}</Link>
            )}

            <div className="pt-3">
              {renderPreferenceControl(mobilePreferenceRef, 'mobile')}
            </div>
          </motion.nav>
        )}
      </header>

      <main className="flex-grow flex flex-col w-full px-4 py-8 md:px-12 md:py-12">
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
