import { lazy, type ComponentType } from 'react';

type RouteComponent = ComponentType<Record<string, never>>;
type RouteModule = { default: RouteComponent };

export type RouteLoader = () => Promise<RouteModule>;

const cacheRoute = (loader: RouteLoader): RouteLoader => {
  let promise: Promise<RouteModule> | null = null;
  return () => {
    if (!promise) {
      promise = loader();
    }
    return promise;
  };
};

export const preloadRoute = (loader?: RouteLoader) => {
  if (!loader) {
    return;
  }
  void loader().catch(() => undefined);
};

export const routeLoaders = {
  articleDetail: cacheRoute(() => import('../pages/ArticleDetail')),
  login: cacheRoute(() => import('../pages/Login')),
  register: cacheRoute(() => import('../pages/Register')),
  forgotPassword: cacheRoute(() => import('../pages/ForgotPassword')),
  archive: cacheRoute(() => import('../pages/Archive')),
  search: cacheRoute(() => import('../pages/Search')),
  ask: cacheRoute(() => import('../pages/Ask')),
  projects: cacheRoute(() => import('../pages/Projects')),
  profile: cacheRoute(() => import('../pages/Profile')),
  adminLayout: cacheRoute(() => import('../pages/admin/AdminLayout')),
  adminArticles: cacheRoute(() => import('../pages/admin/Articles')),
  articleEditor: cacheRoute(() => import('../pages/admin/ArticleEditor')),
  articlePreview: cacheRoute(() => import('../pages/admin/ArticlePreview')),
  articleVersions: cacheRoute(() => import('../pages/admin/ArticleVersions')),
  adminDashboard: cacheRoute(() => import('../pages/admin/Dashboard')),
  adminCategories: cacheRoute(() => import('../pages/admin/Categories')),
  adminTags: cacheRoute(() => import('../pages/admin/Tags')),
  adminUsers: cacheRoute(() => import('../pages/admin/Users')),
  adminLogs: cacheRoute(() => import('../pages/admin/Logs')),
  adminSettings: cacheRoute(() => import('../pages/admin/Settings')),
  adminAISettings: cacheRoute(() => import('../pages/admin/AISettings')),
  adminRAGSettings: cacheRoute(() => import('../pages/admin/RAGSettings')),
  adminProjectsContent: cacheRoute(() => import('../pages/admin/ProjectsContent')),
  adminMedia: cacheRoute(() => import('../pages/admin/Media')),
  adminAnalytics: cacheRoute(() => import('../pages/admin/Analytics')),
  adminSystem: cacheRoute(() => import('../pages/admin/System')),
  notFound: cacheRoute(() => import('../pages/NotFound')),
} satisfies Record<string, RouteLoader>;

export const ArticleDetail = lazy(routeLoaders.articleDetail);
export const Login = lazy(routeLoaders.login);
export const Register = lazy(routeLoaders.register);
export const ForgotPassword = lazy(routeLoaders.forgotPassword);
export const Archive = lazy(routeLoaders.archive);
export const Search = lazy(routeLoaders.search);
export const Ask = lazy(routeLoaders.ask);
export const Projects = lazy(routeLoaders.projects);
export const Profile = lazy(routeLoaders.profile);
export const AdminLayout = lazy(routeLoaders.adminLayout);
export const AdminArticles = lazy(routeLoaders.adminArticles);
export const ArticleEditor = lazy(routeLoaders.articleEditor);
export const ArticlePreview = lazy(routeLoaders.articlePreview);
export const ArticleVersions = lazy(routeLoaders.articleVersions);
export const AdminDashboard = lazy(routeLoaders.adminDashboard);
export const AdminCategories = lazy(routeLoaders.adminCategories);
export const AdminTags = lazy(routeLoaders.adminTags);
export const AdminUsers = lazy(routeLoaders.adminUsers);
export const AdminLogs = lazy(routeLoaders.adminLogs);
export const AdminSettings = lazy(routeLoaders.adminSettings);
export const AdminAISettings = lazy(routeLoaders.adminAISettings);
export const AdminRAGSettings = lazy(routeLoaders.adminRAGSettings);
export const AdminProjectsContent = lazy(routeLoaders.adminProjectsContent);
export const NotFound = lazy(routeLoaders.notFound);
export const AdminMedia = lazy(routeLoaders.adminMedia);
export const AdminAnalytics = lazy(routeLoaders.adminAnalytics);
export const AdminSystem = lazy(routeLoaders.adminSystem);
