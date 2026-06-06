import { useEffect } from 'react';
import { Routes, Route } from 'react-router-dom';
import Layout from './components/Layout';
import Home from './pages/Home';
import ArticleDetail from './pages/ArticleDetail';
import Login from './pages/Login';
import Register from './pages/Register';
import Archive from './pages/Archive';
import Search from './pages/Search';
import Profile from './pages/Profile';
import ProtectedRoute from './components/ProtectedRoute';

import AdminLayout from './pages/admin/AdminLayout';
import AdminArticles from './pages/admin/Articles';
import ArticleEditor from './pages/admin/ArticleEditor';
import AdminCategories from './pages/admin/Categories';
import AdminTags from './pages/admin/Tags';
import AdminUsers from './pages/admin/Users';
import AdminLogs from './pages/admin/Logs';
import AdminSettings from './pages/admin/Settings';
import NotFound from './pages/NotFound';
import { usePreferenceStore } from './store/preferences';
import { useSiteSettingsStore } from './store/siteSettings';
import { useAuthStore } from './store/auth';

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

        <Route path="admin" element={<ProtectedRoute requireAdmin />}>
           <Route element={<AdminLayout />}>
             <Route index element={<AdminArticles />} />
             <Route path="articles" element={<AdminArticles />} />
             <Route path="editor/:id" element={<ArticleEditor />} />
             <Route path="categories" element={<AdminCategories />} />
             <Route path="tags" element={<AdminTags />} />
             <Route path="users" element={<AdminUsers />} />
             <Route path="logs" element={<AdminLogs />} />
             <Route path="settings" element={<AdminSettings />} />
           </Route>
        </Route>
        <Route path="*" element={<NotFound />} />
      </Route>
    </Routes>
  )
}

export default App;
