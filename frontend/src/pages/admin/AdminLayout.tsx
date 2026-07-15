import React from 'react';
import { Link, useLocation, useOutlet } from 'react-router-dom';
import { AnimatePresence, motion, useReducedMotion } from 'framer-motion';
import { PreloadNavLink } from '../../components/PreloadLink';
import { translate, type TranslationKey } from '../../i18n';
import { usePreferenceStore } from '../../store/preferences';
import { useAuthStore } from '../../store/auth';
import { useSiteSettingsStore } from '../../store/siteSettings';
import { routeLoaders } from '../../routes/lazyRoutes';

const navLinkClass = ({ isActive }: { isActive: boolean }) =>
  [
    'relative flex min-h-11 items-center rounded-md px-3 py-2 text-sm font-medium transition-colors duration-fast',
    isActive
      ? 'bg-ochre text-on-accent'
      : 'text-on-dark-soft hover:bg-surface-dark-soft hover:text-on-dark',
  ].join(' ');

/** 路径段 → i18n key 映射（保留未知段为原值显示） */
const segmentLabelKey: Record<string, TranslationKey> = {
  admin: 'admin.breadcrumb.home',
  dashboard: 'admin.dashboard',
  articles: 'admin.articles',
  categories: 'admin.categories',
  tags: 'admin.tags',
  users: 'admin.users',
  logs: 'admin.logs',
  settings: 'admin.settings',
  'ai-settings': 'admin.aiSettings',
  projects: 'admin.projects',
  media: 'admin.media',
  analytics: 'admin.analytics',
  system: 'admin.system',
};

const AdminLayout: React.FC = () => {
  const language = usePreferenceStore((state) => state.language);
  const user = useAuthStore((state) => state.user);
  const projectsPageEnabled = useSiteSettingsStore((state) => state.projectsPageEnabled);
  const location = useLocation();
  const outlet = useOutlet();
  const shouldReduceMotion = useReducedMotion();
  const t = (key: TranslationKey) => translate(language, key);
  const contentTransition = shouldReduceMotion
    ? { duration: 0 }
    : { duration: 0.22, ease: [0.22, 1, 0.36, 1] };
  const contentVariants = shouldReduceMotion
    ? {
        initial: { opacity: 1, y: 0 },
        animate: { opacity: 1, y: 0 },
        exit: { opacity: 1, y: 0 },
      }
    : {
        initial: { opacity: 0, y: 6 },
        animate: { opacity: 1, y: 0 },
        exit: { opacity: 0, y: -3 },
      };

  // 面包屑：基于 pathname 切片
  const breadcrumbSegments = location.pathname
    .split('/')
    .filter(Boolean)
    .reduce<Array<{ key: string; label: string; href: string }>>((acc, segment, idx, all) => {
      const href = '/' + all.slice(0, idx + 1).join('/');
      const labelKey = segmentLabelKey[segment];
      const label = labelKey ? t(labelKey) : segment;
      acc.push({ key: `${segment}-${idx}`, label, href });
      return acc;
    }, []);

  return (
    <div className="editorial-container flex w-full flex-col gap-5 lg:flex-row lg:gap-6">
      <aside className="w-full shrink-0 rounded-xl bg-surface-dark p-4 text-on-dark lg:sticky lg:top-24 lg:min-h-[calc(100vh-8rem)] lg:w-56 lg:self-start lg:p-5">
        <div className="mb-4 flex items-center gap-3 md:mb-8">
          <span aria-hidden="true" className="text-xl text-ochre">✣</span>
          <h2 className="font-display text-2xl text-on-dark">{t('admin.title')}</h2>
        </div>
        <nav className="flex gap-2 overflow-x-auto whitespace-nowrap pb-1 lg:flex-col lg:overflow-visible lg:whitespace-normal lg:pb-0">
          <PreloadNavLink to="/admin/dashboard" preload={routeLoaders.adminDashboard} className={navLinkClass}>
            {t('admin.dashboard')}
          </PreloadNavLink>
          <PreloadNavLink to="/admin/articles" preload={routeLoaders.adminArticles} className={navLinkClass}>
            {t('admin.articles')}
          </PreloadNavLink>
          <PreloadNavLink to="/admin/categories" preload={routeLoaders.adminCategories} className={navLinkClass}>
            {t('admin.categories')}
          </PreloadNavLink>
          <PreloadNavLink to="/admin/tags" preload={routeLoaders.adminTags} className={navLinkClass}>
            {t('admin.tags')}
          </PreloadNavLink>
          <PreloadNavLink to="/admin/media" preload={routeLoaders.adminMedia} className={navLinkClass}>
            {t('admin.media')}
          </PreloadNavLink>
          <PreloadNavLink to="/admin/analytics" preload={routeLoaders.adminAnalytics} className={navLinkClass}>
            {t('admin.analytics')}
          </PreloadNavLink>
          <div className="hidden h-3 lg:block"></div>
          {user?.role === 'admin' && (
            <>
              <PreloadNavLink to="/admin/users" preload={routeLoaders.adminUsers} className={navLinkClass}>
                {t('admin.users')}
              </PreloadNavLink>
              <PreloadNavLink to="/admin/logs" preload={routeLoaders.adminLogs} className={navLinkClass}>
                {t('admin.logs')}
              </PreloadNavLink>
              <PreloadNavLink to="/admin/settings" preload={routeLoaders.adminSettings} className={navLinkClass}>
                {t('admin.settings')}
              </PreloadNavLink>
              <PreloadNavLink to="/admin/ai-settings" preload={routeLoaders.adminAISettings} className={navLinkClass}>
                {t('admin.aiSettings')}
              </PreloadNavLink>
              <PreloadNavLink to="/admin/system" preload={routeLoaders.adminSystem} className={navLinkClass}>
                {t('admin.system')}
              </PreloadNavLink>
              {projectsPageEnabled && (
                <PreloadNavLink to="/admin/projects" preload={routeLoaders.adminProjectsContent} className={navLinkClass}>
                  {t('admin.projects')}
                </PreloadNavLink>
              )}
            </>
          )}
        </nav>
      </aside>

      <div className="admin-workspace min-w-0 flex-grow rounded-xl bg-surface-soft p-4 sm:p-6 lg:p-8">
        {breadcrumbSegments.length > 0 && (
          <nav aria-label={t('common.breadcrumb')} className="mb-7 hidden flex-wrap items-center gap-1.5 text-xs text-muted md:flex">
            {breadcrumbSegments.map((seg, idx) => {
              const isLast = idx === breadcrumbSegments.length - 1;
              return (
                <React.Fragment key={seg.key}>
                  {isLast ? (
                    <span aria-current="page" className="text-ink">{seg.label}</span>
                  ) : (
                    <Link to={seg.href} className="hover:text-ochre transition-colors duration-fast">
                      {seg.label}
                    </Link>
                  )}
                  {!isLast && <span className="opacity-50 select-none">/</span>}
                </React.Fragment>
              );
            })}
          </nav>
        )}
        <AnimatePresence mode="wait" initial={false}>
          <motion.div
            key={location.pathname}
            variants={contentVariants}
            initial="initial"
            animate="animate"
            exit="exit"
            transition={contentTransition}
            className="page-transition-surface min-w-0"
          >
            {outlet}
          </motion.div>
        </AnimatePresence>
      </div>
    </div>
  );
};

export default AdminLayout;
