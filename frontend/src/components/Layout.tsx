import React, { useEffect, useId, useRef, useState } from 'react';
import { Link, useLocation, useNavigate, useOutlet } from 'react-router-dom';
import { AnimatePresence, motion, useReducedMotion } from 'framer-motion';
import { PreloadNavLink } from './PreloadLink';
import Toaster from './Toaster';
import BackToTop from './BackToTop';
import RequestProgressBar from './RequestProgressBar';
import { useAuthStore } from '../store/auth';
import { usePreferenceStore } from '../store/preferences';
import { useSiteSettingsStore } from '../store/siteSettings';
import { useShallow } from 'zustand/react/shallow';
import { logout as apiLogout } from '../api/auth';
import { isHttpAvatarUrl, normalizeAvatarUrl } from '../utils/avatar';
import { formatText, translate } from '../i18n';
import { notifyFromError } from '../utils/notify';
import { toAppError } from '../utils/error';
import { resolveLogoutFailure } from '../store/logoutPolicy';
import { useSEO } from '../utils/seo';
import { routeLoaders } from '../routes/lazyRoutes';
import { trapFocus } from '../utils/focusTrap';
import { routeUsesOwnSEO } from '../utils/routeSeo';

const Layout: React.FC = () => {
  const { user, logout } = useAuthStore(useShallow((state) => ({ user: state.user, logout: state.logout })));
  const {
    language,
    themePreference,
    effectiveTheme,
    accentColor,
    setLanguage,
    setThemePreference,
    setAccentColor,
    resetAccentColor,
  } = usePreferenceStore(
    useShallow((state) => ({
      language: state.language,
      themePreference: state.themePreference,
      effectiveTheme: state.effectiveTheme,
      accentColor: state.accentColor,
      setLanguage: state.setLanguage,
      setThemePreference: state.setThemePreference,
      setAccentColor: state.setAccentColor,
      resetAccentColor: state.resetAccentColor,
    })),
  );
  const { projectsPageEnabled, projectsNavHidden } = useSiteSettingsStore(
    useShallow((state) => ({
      projectsPageEnabled: state.projectsPageEnabled,
      projectsNavHidden: state.projectsNavHidden,
    })),
  );
  const navigate = useNavigate();
  const location = useLocation();
  const outlet = useOutlet();
  const shouldReduceMotion = useReducedMotion();
  const [isPreferenceOpen, setIsPreferenceOpen] = useState(false);
  const [isMobileMenuOpen, setIsMobileMenuOpen] = useState(false);
  const headerRef = useRef<HTMLElement>(null);
  const mobileMenuRef = useRef<HTMLElement>(null);
  const mobileMenuTriggerRef = useRef<HTMLButtonElement>(null);
  const desktopPreferenceRef = useRef<HTMLDivElement>(null);
  const mobilePreferenceRef = useRef<HTMLDivElement>(null);
  const desktopPreferenceTriggerRef = useRef<HTMLButtonElement>(null);
  const mobilePreferenceTriggerRef = useRef<HTMLButtonElement>(null);
  const lastPreferenceTriggerRef = useRef<HTMLButtonElement | null>(null);
  const desktopPreferencePanelId = useId();
  const mobilePreferencePanelId = useId();
  const desktopPreferenceTitleId = useId();
  const mobilePreferenceTitleId = useId();
  const avatarUrl = normalizeAvatarUrl(user?.avatarUrl);
  const shouldShowAvatar = isHttpAvatarUrl(avatarUrl);
  const t = (key: Parameters<typeof translate>[1]) => translate(language, key);
  const currentLanguageLabel = language === 'zh' ? t('preferences.languageZh') : t('preferences.languageEn');
  const currentThemeLabel = themePreference === 'system'
    ? `${t('preferences.themeSystem')} / ${effectiveTheme === 'dark' ? t('preferences.themeDark') : t('preferences.themeLight')}`
    : (effectiveTheme === 'dark' ? t('preferences.themeDark') : t('preferences.themeLight'));
  const pageTransitionKey = location.pathname.startsWith('/admin') ? 'admin' : location.pathname;
  const usesRouteSEO = routeUsesOwnSEO(location.pathname);
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
      if (isMobileMenuOpen && mobileMenuRef.current && event.key === 'Tab') {
        trapFocus(mobileMenuRef.current, event);
        return;
      }
      if (event.key === 'Escape') {
        if (isPreferenceOpen) {
          event.preventDefault();
          setIsPreferenceOpen(false);
          window.requestAnimationFrame(() => lastPreferenceTriggerRef.current?.focus());
          return;
        }

        setIsMobileMenuOpen(false);
        window.requestAnimationFrame(() => mobileMenuTriggerRef.current?.focus());
      }
    };

    document.addEventListener('mousedown', handlePointerDown);
    document.addEventListener('keydown', handleKeyDown);

    return () => {
      document.removeEventListener('mousedown', handlePointerDown);
      document.removeEventListener('keydown', handleKeyDown);
    };
  }, [isPreferenceOpen, isMobileMenuOpen]);

  useEffect(() => {
    if (!isMobileMenuOpen) return undefined;
    const previousOverflow = document.body.style.overflow;
    document.body.style.overflow = 'hidden';
    window.requestAnimationFrame(() => {
      mobileMenuRef.current?.querySelector<HTMLElement>('a, button')?.focus();
    });
    return () => {
      document.body.style.overflow = previousOverflow;
    };
  }, [isMobileMenuOpen]);

  const handleLogout = async () => {
    // refreshToken 走 HttpOnly Cookie，请求体留空；服务端按 Cookie 撤销并清除 Cookie。
    try {
      await apiLogout({});
    } catch (error) {
      const decision = resolveLogoutFailure(toAppError(error).status ?? 0);
      if (!decision.clearSession) {
        notifyFromError(error, 'toast.logoutFailed');
        return;
      }
    }
    logout();
    setIsPreferenceOpen(false);
    setIsMobileMenuOpen(false);
    navigate('/');
  };

  const desktopLinkClass = ({ isActive }: { isActive: boolean }) => [
    'relative flex min-h-11 items-center px-1 font-medium transition-colors after:absolute after:bottom-1 after:left-0 after:h-px after:w-full after:origin-left after:bg-ochre after:transition-transform',
    isActive ? 'text-ink after:scale-x-100' : 'text-muted after:scale-x-0 hover:text-ink',
  ].join(' ');
  const mobileLinkClass = ({ isActive }: { isActive: boolean }) => [
    'flex min-h-14 items-center justify-between border-b border-hairline px-1 py-3 text-left font-display text-3xl transition-colors',
    isActive ? 'text-ochre' : 'text-ink hover:text-ochre',
  ].join(' ');
  const mobileActionClass = 'flex min-h-14 w-full items-center border-b border-hairline px-1 py-3 text-left font-display text-3xl text-ink transition-colors hover:text-ochre';
  const closeMobileMenu = () => setIsMobileMenuOpen(false);

  const renderPreferencePanel = (
    variant: 'desktop' | 'mobile',
    panelId: string,
    titleId: string,
  ) => (
    <motion.div
      id={panelId}
      role="dialog"
      aria-labelledby={titleId}
      initial={{ opacity: 0, y: -6 }}
      animate={{ opacity: 1, y: 0 }}
      transition={{ duration: 0.2, ease: 'easeOut' }}
      className={`z-50 rounded-xl border border-hairline bg-paper p-5 text-left shadow-lg ${
        variant === 'mobile'
          ? 'fixed left-4 right-4 top-24 max-h-[calc(100dvh-7rem-env(safe-area-inset-bottom))] overflow-y-auto pb-[env(safe-area-inset-bottom)] md:hidden'
          : 'absolute right-0 top-full mt-3 hidden w-[min(21rem,calc(100vw-3rem))] md:block'
      }`}
    >
      <div className="border-b border-mountain-grey/60 pb-3">
        <p id={titleId} className="text-sm font-bold tracking-widest text-ink">{t('preferences.title')}</p>
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
          <div className="grid grid-cols-2 overflow-hidden rounded-md border border-hairline bg-surface-soft p-1">
            <button
              type="button"
              onClick={() => setLanguage('zh')}
              className={`min-h-11 rounded-sm px-3 py-2 text-sm transition-colors ${language === 'zh' ? 'bg-surface-dark text-on-dark' : 'text-muted hover:text-ink'}`}
            >
              {t('preferences.languageZh')}
            </button>
            <button
              type="button"
              onClick={() => setLanguage('en')}
              className={`min-h-11 rounded-sm px-3 py-2 text-sm transition-colors ${language === 'en' ? 'bg-surface-dark text-on-dark' : 'text-muted hover:text-ink'}`}
            >
              {t('preferences.languageEn')}
            </button>
          </div>
        </div>

        <div>
          <p className="mb-2 text-xs tracking-widest text-ink-light">{t('preferences.themeTitle')}</p>
          <div className="grid grid-cols-3 overflow-hidden rounded-md border border-hairline bg-surface-soft p-1">
            <button
              type="button"
              aria-label={t('toggle.themeToLight')}
              onClick={() => setThemePreference('light')}
              className={`min-h-11 rounded-sm px-2 py-2 text-sm transition-colors ${themePreference === 'light' ? 'bg-surface-dark text-on-dark' : 'text-muted hover:text-ink'}`}
            >
              {t('preferences.themeLight')}
            </button>
            <button
              type="button"
              aria-label={t('toggle.themeToDark')}
              onClick={() => setThemePreference('dark')}
              className={`min-h-11 rounded-sm px-2 py-2 text-sm transition-colors ${themePreference === 'dark' ? 'bg-surface-dark text-on-dark' : 'text-muted hover:text-ink'}`}
            >
              {t('preferences.themeDark')}
            </button>
            <button
              type="button"
              onClick={() => setThemePreference('system')}
              className={`min-h-11 rounded-sm px-2 py-2 text-sm transition-colors ${themePreference === 'system' ? 'bg-surface-dark text-on-dark' : 'text-muted hover:text-ink'}`}
            >
              {t('preferences.themeSystem')}
            </button>
          </div>
        </div>

        <div>
          <p className="mb-2 text-xs tracking-widest text-ink-light">{t('preferences.accentTitle')}</p>
          <div className="flex items-center gap-3 rounded-md border border-hairline bg-surface-soft px-3 py-2">
            <input
              type="color"
              value={accentColor || '#cc785c'}
              onChange={(event) => setAccentColor(event.target.value)}
              aria-label={t('preferences.accentTitle')}
              className="h-8 w-10 cursor-pointer border-0 bg-transparent p-0"
            />
            <span className="min-w-0 flex-1 text-xs tracking-[0.14em] text-ink-light">
              {accentColor || t('preferences.accentDefault')}
            </span>
            <button
              type="button"
              onClick={resetAccentColor}
              className="shrink-0 text-xs tracking-widest text-ochre hover:text-ink"
            >
              {t('preferences.accentReset')}
            </button>
          </div>
        </div>
      </div>
    </motion.div>
  );

  const renderPreferenceControl = (
    ref: React.RefObject<HTMLDivElement>,
    triggerRef: React.RefObject<HTMLButtonElement>,
    variant: 'desktop' | 'mobile',
    panelId: string,
    titleId: string,
  ) => (
    <div ref={ref} className="relative">
      <button
        ref={triggerRef}
        type="button"
        aria-haspopup="dialog"
        aria-expanded={isPreferenceOpen}
        aria-controls={panelId}
        aria-label={t('preferences.openLabel')}
        onClick={() => setIsPreferenceOpen((open) => {
          if (!open) {
            lastPreferenceTriggerRef.current = triggerRef.current;
          }
          return !open;
        })}
        className={`group flex min-h-11 items-center gap-2 rounded-md border border-hairline bg-paper text-ink transition-colors hover:border-ink ${
          variant === 'mobile' ? 'w-full justify-between px-3 py-3' : 'px-3 py-2'
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

      {isPreferenceOpen && renderPreferencePanel(variant, panelId, titleId)}
    </div>
  );

  return (
    <div className="flex min-h-screen flex-col bg-paper font-sans text-ink transition-colors duration-page">
      <Toaster />
      <RequestProgressBar />
      <header ref={headerRef} className="sticky top-0 z-[95] h-16 border-b border-hairline bg-[var(--paper-muted)] px-4 backdrop-blur-xl md:px-8 lg:px-12">
        <div className="editorial-container flex h-full items-center justify-between gap-4">
          <Link to="/" className="group flex min-h-11 items-center gap-3 text-ink transition-colors hover:text-ochre">
            <span aria-hidden="true" className="relative flex h-7 w-7 items-center justify-center rounded-full bg-surface-dark text-sm text-on-dark transition-transform duration-base group-hover:rotate-12">✣</span>
            <span className="font-display text-2xl leading-none tracking-[-0.02em] md:text-[1.75rem]">
              {t('brand.name')}
            </span>
          </Link>

          <button
            ref={mobileMenuTriggerRef}
            type="button"
            aria-expanded={isMobileMenuOpen}
            aria-label={isMobileMenuOpen ? t('nav.menuClose') : t('nav.menuOpen')}
            onClick={() => {
              setIsMobileMenuOpen((open) => !open);
              setIsPreferenceOpen(false);
            }}
            className="relative flex h-11 w-11 shrink-0 items-center justify-center rounded-md border border-hairline bg-paper text-ink transition-colors hover:border-ink md:hidden"
          >
            <span className={`absolute h-px w-5 bg-current transition-transform ${isMobileMenuOpen ? 'rotate-45' : '-translate-y-1.5'}`}></span>
            <span className={`absolute h-px w-5 bg-current transition-opacity ${isMobileMenuOpen ? 'opacity-0' : 'opacity-100'}`}></span>
            <span className={`absolute h-px w-5 bg-current transition-transform ${isMobileMenuOpen ? '-rotate-45' : 'translate-y-1.5'}`}></span>
          </button>

          <nav className="hidden items-center justify-end gap-x-5 text-sm md:flex lg:gap-x-7">
            <PreloadNavLink to="/" end className={desktopLinkClass}>{t('nav.home')}</PreloadNavLink>
            <PreloadNavLink to="/archive" preload={routeLoaders.archive} className={desktopLinkClass}>{t('nav.archive')}</PreloadNavLink>
            <PreloadNavLink to="/search" preload={routeLoaders.search} className={desktopLinkClass}>{t('nav.search')}</PreloadNavLink>
            {projectsPageEnabled && !projectsNavHidden && (
              <PreloadNavLink to="/projects" preload={routeLoaders.projects} className={desktopLinkClass}>{t('nav.projects')}</PreloadNavLink>
            )}
            {user ? (
              <>
                {(user.role === 'admin' || user.role === 'editor') && (
                  <PreloadNavLink to="/admin" preload={[routeLoaders.adminLayout, routeLoaders.adminDashboard]} className={({ isActive }) => `transition-colors font-bold ${isActive ? 'text-ochre' : 'hover:text-ochre'}`}>{t('nav.admin')}</PreloadNavLink>
                )}
                <PreloadNavLink to="/profile" preload={routeLoaders.profile} className={({ isActive }) => `group flex items-center space-x-2 transition-colors ${isActive ? 'font-bold text-ochre' : 'hover:text-ink'}`}>
                  {shouldShowAvatar && (
                    <img src={avatarUrl} alt="avatar" decoding="async" className="w-6 h-6 rounded-full border border-mountain-grey group-hover:border-ochre transition-colors object-cover" />
                  )}
                  <span>{user.nickname || user.account}</span>
                </PreloadNavLink>
                <button type="button" className="hover:text-ochre transition-colors" onClick={handleLogout}>
                  {t('nav.logout')}
                </button>
              </>
            ) : (
              <PreloadNavLink to="/login" preload={routeLoaders.login} className={desktopLinkClass}>{t('nav.login')}</PreloadNavLink>
            )}

            {renderPreferenceControl(
              desktopPreferenceRef,
              desktopPreferenceTriggerRef,
              'desktop',
              desktopPreferencePanelId,
              desktopPreferenceTitleId,
            )}
          </nav>
        </div>

        {isMobileMenuOpen && (
          <motion.nav
            ref={mobileMenuRef}
            tabIndex={-1}
            aria-label={t('nav.menuOpen')}
            initial={{ y: -6 }}
            animate={{ y: 0 }}
            transition={{ duration: 0.18, ease: 'easeOut' }}
            className="absolute inset-x-0 top-full z-[94] h-[calc(100dvh-4rem)] overflow-y-auto border-t border-hairline bg-paper px-5 pb-[calc(2rem+env(safe-area-inset-bottom))] pt-7 md:hidden"
          >
            <PreloadNavLink to="/" end onClick={closeMobileMenu} className={mobileLinkClass}>{t('nav.home')}</PreloadNavLink>
            <PreloadNavLink to="/archive" preload={routeLoaders.archive} onClick={closeMobileMenu} className={mobileLinkClass}>{t('nav.archive')}</PreloadNavLink>
            <PreloadNavLink to="/search" preload={routeLoaders.search} onClick={closeMobileMenu} className={mobileLinkClass}>{t('nav.search')}</PreloadNavLink>
            {projectsPageEnabled && !projectsNavHidden && (
              <PreloadNavLink to="/projects" preload={routeLoaders.projects} onClick={closeMobileMenu} className={mobileLinkClass}>{t('nav.projects')}</PreloadNavLink>
            )}
            {user ? (
              <>
                {(user.role === 'admin' || user.role === 'editor') && (
                  <PreloadNavLink to="/admin" preload={[routeLoaders.adminLayout, routeLoaders.adminDashboard]} onClick={closeMobileMenu} className={({ isActive }) => `${mobileLinkClass({ isActive })} font-bold`}>
                    {t('nav.admin')}
                  </PreloadNavLink>
                )}
                <PreloadNavLink to="/profile" preload={routeLoaders.profile} onClick={closeMobileMenu} className={({ isActive }) => `${mobileLinkClass({ isActive })} flex items-center gap-3`}>
                  {shouldShowAvatar && (
                    <img src={avatarUrl} alt="avatar" loading="lazy" decoding="async" className="h-7 w-7 rounded-full border border-mountain-grey object-cover" />
                  )}
                  <span>{user.nickname || user.account}</span>
                </PreloadNavLink>
                <button type="button" className={mobileActionClass} onClick={handleLogout}>
                  {t('nav.logout')}
                </button>
              </>
            ) : (
              <PreloadNavLink to="/login" preload={routeLoaders.login} onClick={closeMobileMenu} className={mobileLinkClass}>{t('nav.login')}</PreloadNavLink>
            )}

            <div className="pt-6">
              {renderPreferenceControl(
                mobilePreferenceRef,
                mobilePreferenceTriggerRef,
                'mobile',
                mobilePreferencePanelId,
                mobilePreferenceTitleId,
              )}
            </div>
          </motion.nav>
        )}
      </header>

      <a
        href="#main-content"
        className="sr-only focus-visible:not-sr-only focus-visible:fixed focus-visible:top-3 focus-visible:left-1/2 focus-visible:-translate-x-1/2 focus-visible:z-[200] focus-visible:border focus-visible:border-ochre focus-visible:bg-paper focus-visible:px-4 focus-visible:py-2 focus-visible:text-sm focus-visible:tracking-widest focus-visible:text-ink focus-visible:shadow-md"
      >
        {t('a11y.skipToContent')}
      </a>
      <main id="main-content" tabIndex={-1} className="flex w-full flex-grow flex-col px-4 py-8 md:px-8 md:py-12 lg:px-12 lg:py-16">
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

      <footer className="mt-8 bg-surface-dark px-4 py-14 text-on-dark-soft md:px-8 md:py-16 lg:px-12">
        <div className="editorial-container grid gap-10 md:grid-cols-[1.4fr_1fr] md:items-end">
          <div>
            <p className="font-display text-4xl leading-tight text-on-dark md:text-5xl">{t('brand.nameEn')}</p>
            <p className="mt-4 max-w-lg text-sm leading-relaxed">{t('footer.poem')} {t('footer.crafted')}</p>
          </div>
          <nav className="grid grid-cols-2 gap-x-6 gap-y-3 text-sm md:justify-self-end md:text-right" aria-label="Footer">
            <Link to="/" className="hover:text-on-dark">{t('nav.home')}</Link>
            <Link to="/archive" className="hover:text-on-dark">{t('nav.archive')}</Link>
            <Link to="/search" className="hover:text-on-dark">{t('nav.search')}</Link>
            <Link to="/login" className="hover:text-on-dark">{t('nav.login')}</Link>
          </nav>
          <p className="border-t border-white/10 pt-6 text-xs md:col-span-2">&copy; 2026 {t('brand.nameEn')}</p>
        </div>
      </footer>

      <BackToTop />
    </div>
  );
};

export default Layout;
