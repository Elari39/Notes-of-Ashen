import { Suspense, lazy, useEffect, useRef, type ReactElement } from 'react';
import { Routes, Route, useLocation } from 'react-router-dom';
import Layout from './components/Layout';
import Home from './pages/Home';
import ProtectedRoute from './components/ProtectedRoute';
import { usePreferenceStore } from './store/preferences';
import { useSiteSettingsStore } from './store/siteSettings';
import { useAuthStore } from './store/auth';
import { reportVisit } from './api/traffic';

const ArticleDetail = lazy(() => import('./pages/ArticleDetail'));
const Login = lazy(() => import('./pages/Login'));
const Register = lazy(() => import('./pages/Register'));
const ForgotPassword = lazy(() => import('./pages/ForgotPassword'));
const Archive = lazy(() => import('./pages/Archive'));
const Search = lazy(() => import('./pages/Search'));
const Resume = lazy(() => import('./pages/Resume'));
const Projects = lazy(() => import('./pages/Projects'));
const Profile = lazy(() => import('./pages/Profile'));
const AdminLayout = lazy(() => import('./pages/admin/AdminLayout'));
const AdminArticles = lazy(() => import('./pages/admin/Articles'));
const ArticleEditor = lazy(() => import('./pages/admin/ArticleEditor'));
const ArticlePreview = lazy(() => import('./pages/admin/ArticlePreview'));
const ArticleVersions = lazy(() => import('./pages/admin/ArticleVersions'));
const AdminDashboard = lazy(() => import('./pages/admin/Dashboard'));
const AdminCategories = lazy(() => import('./pages/admin/Categories'));
const AdminTags = lazy(() => import('./pages/admin/Tags'));
const AdminUsers = lazy(() => import('./pages/admin/Users'));
const AdminLogs = lazy(() => import('./pages/admin/Logs'));
const AdminSettings = lazy(() => import('./pages/admin/Settings'));
const AdminResumeContent = lazy(() => import('./pages/admin/ResumeContent'));
const AdminProjectsContent = lazy(() => import('./pages/admin/ProjectsContent'));
const NotFound = lazy(() => import('./pages/NotFound'));

const RouteLoading = () => (
  <main className="min-h-[60vh] px-6 py-24 text-center text-sm tracking-[0.24em] text-ink-light">
    LOADING
  </main>
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
    <Suspense fallback={<RouteLoading />}>
      <TrafficReporter />
      <Routes>
        <Route path="/" element={<Layout />}>
          <Route index element={<Home />} />
          <Route path="article/:id" element={<ArticleDetail />} />
          <Route path="archive" element={<Archive />} />
          <Route path="search" element={<Search />} />
          <Route
            path="resume"
            element={(
              <PublicFeatureRoute enabled={resumePageEnabled}>
                <Resume />
              </PublicFeatureRoute>
            )}
          />
          <Route
            path="projects"
            element={(
              <PublicFeatureRoute enabled={projectsPageEnabled}>
                <Projects />
              </PublicFeatureRoute>
            )}
          />
          <Route path="login" element={<Login />} />
          <Route path="register" element={<Register />} />
          <Route path="forgot-password" element={<ForgotPassword />} />

          <Route element={<ProtectedRoute />}>
            <Route path="profile" element={<Profile />} />
          </Route>

          <Route path="admin" element={<ProtectedRoute allowedRoles={['editor', 'admin']} />}>
            <Route element={<AdminLayout />}>
              <Route index element={<AdminDashboard />} />
              <Route path="dashboard" element={<AdminDashboard />} />
              <Route path="articles" element={<AdminArticles />} />
              <Route path="editor/:id" element={<ArticleEditor />} />
              <Route path="preview/:id" element={<ArticlePreview />} />
              <Route path="articles/:id/versions" element={<ArticleVersions />} />
              <Route path="categories" element={<AdminCategories />} />
              <Route path="tags" element={<AdminTags />} />
              <Route element={<ProtectedRoute allowedRoles={['admin']} />}>
                <Route path="users" element={<AdminUsers />} />
                <Route path="logs" element={<AdminLogs />} />
                <Route path="settings" element={<AdminSettings />} />
                <Route path="resume" element={<AdminResumeContent />} />
                <Route path="projects" element={<AdminProjectsContent />} />
              </Route>
            </Route>
          </Route>
          <Route path="*" element={<NotFound />} />
        </Route>
      </Routes>
    </Suspense>
  )
}

export default App;

const ignoredTrafficPrefixes = ['/admin', '/login', '/register', '/profile', '/forgot-password'];

const PublicFeatureRoute = ({ enabled, children }: { enabled: boolean; children: ReactElement }) => {
  const { hasLoaded, error } = useSiteSettingsStore();

  if (!hasLoaded) {
    return <RouteLoading />;
  }

  if (error || !enabled) {
    return <NotFound />;
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
