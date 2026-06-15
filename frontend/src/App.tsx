import { Suspense, useEffect, useRef, type ReactElement } from 'react';
import { Routes, Route, useLocation } from 'react-router-dom';
import Layout from './components/Layout';
import Home from './pages/Home';
import ProtectedRoute from './components/ProtectedRoute';
import PagePendingState, { RoutePendingIndicator } from './components/RoutePending';
import { usePreferenceStore } from './store/preferences';
import { useSiteSettingsStore } from './store/siteSettings';
import { useAuthStore } from './store/auth';
import { reportVisit } from './api/traffic';
import {
  AdminAISettings,
  AdminArticles,
  AdminCategories,
  AdminDashboard,
  AdminLayout,
  AdminLogs,
  AdminProjectsContent,
  AdminResumeContent,
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
  Profile,
  Projects,
  Register,
  Resume,
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
  const {
    fetchSettings: fetchSiteSettings,
    resumePageEnabled,
    projectsPageEnabled,
  } = useSiteSettingsStore();
  const initializeAuth = useAuthStore((state) => state.initializeAuth);

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
    <>
      <TrafficReporter />
      <Routes>
        <Route path="/" element={<Layout />}>
          <Route index element={<Home />} />
          <Route path="article/:id" element={withRouteSuspense(<ArticleDetail />)} />
          <Route path="archive" element={withRouteSuspense(<Archive />)} />
          <Route path="search" element={withRouteSuspense(<Search />)} />
          <Route
            path="resume"
            element={(
              <PublicFeatureRoute enabled={resumePageEnabled}>
                {withRouteSuspense(<Resume />)}
              </PublicFeatureRoute>
            )}
          />
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
              <Route element={<ProtectedRoute allowedRoles={['admin']} />}>
                <Route path="users" element={withRouteSuspense(<AdminUsers />, 'admin')} />
                <Route path="logs" element={withRouteSuspense(<AdminLogs />, 'admin')} />
                <Route path="settings" element={withRouteSuspense(<AdminSettings />, 'admin')} />
                <Route path="ai-settings" element={withRouteSuspense(<AdminAISettings />, 'admin')} />
                <Route path="resume" element={withRouteSuspense(<AdminResumeContent />, 'admin')} />
                <Route path="projects" element={withRouteSuspense(<AdminProjectsContent />, 'admin')} />
              </Route>
            </Route>
          </Route>
          <Route path="*" element={withRouteSuspense(<NotFound />)} />
        </Route>
      </Routes>
    </>
  )
}

export default App;

const ignoredTrafficPrefixes = ['/admin', '/login', '/register', '/profile', '/forgot-password'];

const PublicFeatureRoute = ({ enabled, children }: { enabled: boolean; children: ReactElement }) => {
  const { hasLoaded, error } = useSiteSettingsStore();

  if (!hasLoaded) {
    return <PagePendingState />;
  }

  if (error || !enabled) {
    return withRouteSuspense(<NotFound />);
  }

  return children;
};

const TrafficReporter = () => {
  const location = useLocation();
  const previousPublicPath = useRef('');
  const {
    hasLoaded: siteSettingsLoaded,
    resumePageEnabled,
    projectsPageEnabled,
  } = useSiteSettingsStore();

  useEffect(() => {
    const path = `${location.pathname}${location.search}`;
    if (ignoredTrafficPrefixes.some((prefix) => location.pathname === prefix || location.pathname.startsWith(`${prefix}/`))) {
      return;
    }
    if (isDisabledFeaturePath(location.pathname, siteSettingsLoaded, resumePageEnabled, projectsPageEnabled)) {
      return;
    }
    const duplicateKey = `traffic:${path}`;
    const lastReported = Number(sessionStorage.getItem(duplicateKey) || 0);
    if (Date.now() - lastReported < 5000) {
      return;
    }
    sessionStorage.setItem(duplicateKey, String(Date.now()));

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
  }, [location.pathname, location.search, projectsPageEnabled, resumePageEnabled, siteSettingsLoaded]);

  return null;
};

const isDisabledFeaturePath = (
  pathname: string,
  siteSettingsLoaded: boolean,
  resumePageEnabled: boolean,
  projectsPageEnabled: boolean,
) => {
  if (!siteSettingsLoaded) {
    return pathname === '/resume' || pathname === '/projects';
  }
  return (pathname === '/resume' && !resumePageEnabled) || (pathname === '/projects' && !projectsPageEnabled);
};
