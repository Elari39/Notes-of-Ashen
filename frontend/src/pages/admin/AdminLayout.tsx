import React from 'react';
import { Link, useLocation, useOutlet } from 'react-router-dom';
import { AnimatePresence, motion, useReducedMotion } from 'framer-motion';
import { PreloadNavLink } from '../../components/PreloadLink';
import Tag from '../../components/ui/Tag';
import { translate, type TranslationKey } from '../../i18n';
import { usePreferenceStore } from '../../store/preferences';
import { useAuthStore } from '../../store/auth';
import { useSiteSettingsStore } from '../../store/siteSettings';
import { useShallow } from 'zustand/react/shallow';
import { routeLoaders } from '../../routes/lazyRoutes';

const navLinkClass = ({ isActive }: { isActive: boolean }) =>
  [
    'relative block py-1 pl-3 transition-colors duration-fast md:py-2',
    isActive
      ? 'text-ochre font-bold md:before:absolute md:before:left-0 md:before:top-1/2 md:before:h-4 md:before:w-[2px] md:before:-translate-y-1/2 md:before:bg-ochre'
      : 'text-ink-light hover:text-ochre',
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
  resume: 'admin.resume',
  projects: 'admin.projects',
};

const AdminLayout: React.FC = () => {
  const language = usePreferenceStore((state) => state.language);
  const user = useAuthStore((state) => state.user);
  const { resumePageEnabled, projectsPageEnabled } = useSiteSettingsStore(
    useShallow((state) => ({
      resumePageEnabled: state.resumePageEnabled,
      projectsPageEnabled: state.projectsPageEnabled,
    })),
  );
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
    <div className="flex flex-col gap-6 md:flex-row md:gap-8 max-w-7xl mx-auto w-full">
      <aside className="w-full shrink-0 border-b border-mountain-grey pb-4 md:w-48 md:border-b-0 md:border-r md:min-h-[60vh] md:pb-0 md:pr-6">
        <div className="mb-4 flex items-center gap-3 md:mb-8">
          <h2 className="text-xl font-bold text-ink tracking-widest">{t('admin.title')}</h2>
          <Tag tone="ochre" size="sm">ADMIN</Tag>
        </div>
        <nav className="flex gap-4 overflow-x-auto whitespace-nowrap pb-1 text-sm tracking-widest md:flex-col md:gap-0 md:space-y-2 md:overflow-visible md:whitespace-normal md:pb-0">
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
          <div className="hidden h-4 md:block"></div>
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
              {resumePageEnabled && (
                <PreloadNavLink to="/admin/resume" preload={routeLoaders.adminResumeContent} className={navLinkClass}>
                  {t('admin.resume')}
                </PreloadNavLink>
              )}
              {projectsPageEnabled && (
                <PreloadNavLink to="/admin/projects" preload={routeLoaders.adminProjectsContent} className={navLinkClass}>
                  {t('admin.projects')}
                </PreloadNavLink>
              )}
            </>
          )}
        </nav>
      </aside>

      <div className="flex-grow min-w-0">
        {breadcrumbSegments.length > 0 && (
          <nav aria-label={t('common.breadcrumb')} className="mb-6 hidden flex-wrap items-center gap-1.5 text-xs tracking-widest text-ink-light md:flex">
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
