import { Suspense, lazy, useEffect } from 'react';
import { Routes, Route } from 'react-router-dom';
import Layout from './components/Layout';
import Home from './pages/Home';
import ProtectedRoute from './components/ProtectedRoute';
import { usePreferenceStore } from './store/preferences';
import { useSiteSettingsStore } from './store/siteSettings';
import { useAuthStore } from './store/auth';

const ArticleDetail = lazy(() => import('./pages/ArticleDetail'));
const Login = lazy(() => import('./pages/Login'));
const Register = lazy(() => import('./pages/Register'));
const Archive = lazy(() => import('./pages/Archive'));
const Search = lazy(() => import('./pages/Search'));
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
const NotFound = lazy(() => import('./pages/NotFound'));

const RouteLoading = () => (
  <main className="min-h-[60vh] px-6 py-24 text-center text-sm tracking-[0.24em] text-ink-light">
    LOADING
  </main>
);

function App() {
  const initializePreferences = usePreferenceStore((state) => state.initializePreferences);
  const fetchSiteSettings = useSiteSettingsStore((state) => state.fetchSettings);
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
      <Routes>
        <Route path="/" element={<Layout />}>
          <Route index element={<Home />} />
          <Route path="article/:id" element={<ArticleDetail />} />
          <Route path="archive" element={<Archive />} />
          <Route path="search" element={<Search />} />
          <Route path="login" element={<Login />} />
          <Route path="register" element={<Register />} />

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
