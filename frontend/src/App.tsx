import { Suspense, useEffect, useRef, type ReactElement } from 'react';
import { Routes, Route, useLocation, useNavigate } from 'react-router-dom';
import Layout from './components/Layout';
import Home from './pages/Home';
import ProtectedRoute from './components/ProtectedRoute';
import PagePendingState, { RoutePendingIndicator } from './components/RoutePending';
import ScrollRestoration from './components/ScrollRestoration';
import { usePreferenceStore } from './store/preferences';
import { useSiteSettingsStore } from './store/siteSettings';
import { useAuthStore } from './store/auth';
import { useShallow } from 'zustand/react/shallow';
import { TooltipProvider } from './components/ui/Tooltip';
import ConfirmDialogHost from './components/ui/ConfirmDialogHost';
import InlineNotice from './components/InlineNotice';
import Button from './components/ui/Button';
import { reportVisit } from './api/traffic';
import { translate } from './i18n';
import { resolvePublicFeatureRoute } from './store/siteSettingsPolicy';
import { safeSessionStorage } from './utils/storage';
import {
  AdminAISettings,
  AdminArticles,
  AdminCategories,
  AdminDashboard,
  AdminLayout,
  AdminLogs,
  AdminProjectsContent,
  AdminSettings,
  AdminTags,
  AdminUsers,
  Archive,
  ArticleDetail,
  ArticleEditor,
  ArticlePreview,
  ArticleVersions,
  ForgotPassword,
  Login,
  NotFound,
  AdminMedia,
  AdminAnalytics,
  AdminSystem,
  Profile,
  Projects,
  Register,
  Search,
} from './routes/lazyRoutes';

type SuspenseVariant = 'page' | 'admin';

const RouteSuspenseFallback = ({ variant }: { variant: SuspenseVariant }) => (
  <>
    <RoutePendingIndicator />
    <PagePendingState variant={variant} />
  </>
);

const withRouteSuspense = (element: ReactElement, variant: SuspenseVariant = 'page') => (
  <Suspense fallback={<RouteSuspenseFallback variant={variant} />}>
    {element}
  </Suspense>
);

function App() {
  const initializePreferences = usePreferenceStore((state) => state.initializePreferences);
  const { fetchSettings: fetchSiteSettings, projectsPageEnabled } = useSiteSettingsStore(
    useShallow((state) => ({
      fetchSettings: state.fetchSettings,
      projectsPageEnabled: state.projectsPageEnabled,
    })),
  );
  const initializeAuth = useAuthStore((state) => state.initializeAuth);
  const setSessionExpiredHandler = useAuthStore((state) => state.setSessionExpiredHandler);
  const navigate = useNavigate();
  const location = useLocation();
  // 用 ref 持有最新 location，避免每次路由变化都重建 sessionExpiredHandler 造成竞态窗口。
  const locationRef = useRef(location);
  locationRef.current = location;

  // 会话失效时跳转登录页并携带来源路径，登录后可回跳
  useEffect(() => {
    setSessionExpiredHandler(() => {
      const current = locationRef.current;
      const from = `${current.pathname}${current.search}${current.hash}`;
      navigate('/login', { state: { from }, replace: true });
    });
    return () => setSessionExpiredHandler(undefined);
  }, [setSessionExpiredHandler, navigate]);

  useEffect(() => {
    return initializePreferences();
  }, [initializePreferences]);

  useEffect(() => {
    fetchSiteSettings();
  }, [fetchSiteSettings]);

  useEffect(() => {
    initializeAuth();
  }, [initializeAuth]);

  return (
    <TooltipProvider delayDuration={300}>
      <ScrollRestoration />
      <TrafficReporter />
      <Routes>
        <Route path="/" element={<Layout />}>
          <Route index element={<Home />} />
          <Route path="article/:id" element={withRouteSuspense(<ArticleDetail />)} />
          <Route path="archive" element={withRouteSuspense(<Archive />)} />
          <Route path="search" element={withRouteSuspense(<Search />)} />
          <Route
            path="projects"
            element={(
              <PublicFeatureRoute enabled={projectsPageEnabled}>
                {withRouteSuspense(<Projects />)}
              </PublicFeatureRoute>
            )}
          />
          <Route path="login" element={withRouteSuspense(<Login />)} />
          <Route path="register" element={withRouteSuspense(<Register />)} />
          <Route path="forgot-password" element={withRouteSuspense(<ForgotPassword />)} />

          <Route element={<ProtectedRoute />}>
            <Route path="profile" element={withRouteSuspense(<Profile />)} />
          </Route>

          <Route path="admin" element={<ProtectedRoute allowedRoles={['editor', 'admin']} />}>
            <Route element={withRouteSuspense(<AdminLayout />, 'admin')}>
              <Route index element={withRouteSuspense(<AdminDashboard />, 'admin')} />
              <Route path="dashboard" element={withRouteSuspense(<AdminDashboard />, 'admin')} />
              <Route path="articles" element={withRouteSuspense(<AdminArticles />, 'admin')} />
              <Route path="editor/:id" element={withRouteSuspense(<ArticleEditor />, 'admin')} />
              <Route path="preview/:id" element={withRouteSuspense(<ArticlePreview />, 'admin')} />
              <Route path="articles/:id/versions" element={withRouteSuspense(<ArticleVersions />, 'admin')} />
              <Route path="categories" element={withRouteSuspense(<AdminCategories />, 'admin')} />
              <Route path="tags" element={withRouteSuspense(<AdminTags />, 'admin')} />
              <Route path="media" element={withRouteSuspense(<AdminMedia />, 'admin')} />
              <Route path="analytics" element={withRouteSuspense(<AdminAnalytics />, 'admin')} />
              <Route element={<ProtectedRoute allowedRoles={['admin']} />}>
                <Route path="users" element={withRouteSuspense(<AdminUsers />, 'admin')} />
                <Route path="logs" element={withRouteSuspense(<AdminLogs />, 'admin')} />
                <Route path="settings" element={withRouteSuspense(<AdminSettings />, 'admin')} />
                <Route path="ai-settings" element={withRouteSuspense(<AdminAISettings />, 'admin')} />
                <Route path="projects" element={withRouteSuspense(<AdminProjectsContent />, 'admin')} />
                <Route path="system" element={withRouteSuspense(<AdminSystem />, 'admin')} />
              </Route>
            </Route>
          </Route>
          <Route path="*" element={withRouteSuspense(<NotFound />)} />
        </Route>
      </Routes>
      <ConfirmDialogHost />
    </TooltipProvider>
  )
}

export default App;

const ignoredTrafficPrefixes = ['/admin', '/login', '/register', '/profile', '/forgot-password'];

const PublicFeatureRoute = ({ enabled, children }: { enabled: boolean; children: ReactElement }) => {
  const language = usePreferenceStore((state) => state.language);
  const { hasLoaded, isLoading, loadError, fetchSettings } = useSiteSettingsStore(
    useShallow((state) => ({
      hasLoaded: state.hasLoaded,
      isLoading: state.isLoading,
      loadError: state.loadError,
      fetchSettings: state.fetchSettings,
    })),
  );
  const routeState = resolvePublicFeatureRoute({ hasLoaded, isLoading, loadError, enabled });

  if (routeState === 'loading') {
    return <PagePendingState />;
  }

  if (routeState === 'error') {
    return (
      <div className="mx-auto w-full max-w-3xl px-4 py-12">
        <InlineNotice
          message={translate(language, 'siteSettings.loadError')}
          action={(
            <Button size="sm" onClick={() => void fetchSettings()} loading={isLoading}>
              {translate(language, 'common.retry')}
            </Button>
          )}
        />
      </div>
    );
  }

  if (routeState === 'notFound') {
    return withRouteSuspense(<NotFound />);
  }

  return children;
};

const TrafficReporter = () => {
  const location = useLocation();
  const previousPublicPath = useRef('');
  const { hasLoaded: siteSettingsLoaded, projectsPageEnabled } = useSiteSettingsStore(
    useShallow((state) => ({
      hasLoaded: state.hasLoaded,
      projectsPageEnabled: state.projectsPageEnabled,
    })),
  );

  useEffect(() => {
    const path = `${location.pathname}${location.search}`;
    if (ignoredTrafficPrefixes.some((prefix) => location.pathname === prefix || location.pathname.startsWith(`${prefix}/`))) {
      return;
    }
    if (isDisabledFeaturePath(location.pathname, siteSettingsLoaded, projectsPageEnabled)) {
      return;
    }
    const duplicateKey = `traffic:${path}`;
    const lastReported = Number(safeSessionStorage.getItem(duplicateKey) || 0);
    if (Date.now() - lastReported < 5000) {
      return;
    }
    safeSessionStorage.setItem(duplicateKey, String(Date.now()));

    const articleMatch = location.pathname.match(/^\/article\/(\d+)/);
    const referrer = previousPublicPath.current
      ? `${window.location.origin}${previousPublicPath.current}`
      : document.referrer;

    previousPublicPath.current = path;
    reportVisit({
      path,
      routeType: articleMatch ? 'article' : 'page',
      articleId: articleMatch ? Number(articleMatch[1]) : undefined,
      referrer,
    }).catch(() => undefined);
  }, [location.pathname, location.search, projectsPageEnabled, siteSettingsLoaded]);

  return null;
};

const isDisabledFeaturePath = (
  pathname: string,
  siteSettingsLoaded: boolean,
  projectsPageEnabled: boolean,
) => {
  if (!siteSettingsLoaded) {
    return pathname === '/projects';
  }
  return pathname === '/projects' && !projectsPageEnabled;
};
