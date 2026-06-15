import React from 'react';
import { useLocation, useOutlet } from 'react-router-dom';
import { AnimatePresence, motion, useReducedMotion } from 'framer-motion';
import { PreloadNavLink } from '../../components/PreloadLink';
import { translate } from '../../i18n';
import { usePreferenceStore } from '../../store/preferences';
import { useAuthStore } from '../../store/auth';
import { useSiteSettingsStore } from '../../store/siteSettings';
import { routeLoaders } from '../../routes/lazyRoutes';

const AdminLayout: React.FC = () => {
  const language = usePreferenceStore((state) => state.language);
  const user = useAuthStore((state) => state.user);
  const { resumePageEnabled, projectsPageEnabled } = useSiteSettingsStore();
  const location = useLocation();
  const outlet = useOutlet();
  const shouldReduceMotion = useReducedMotion();
  const t = (key: Parameters<typeof translate>[1]) => translate(language, key);
  const contentTransition = shouldReduceMotion
    ? { duration: 0.01 }
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

  return (
    <div className="flex flex-col gap-6 md:flex-row md:gap-8 max-w-7xl mx-auto w-full">
      <aside className="w-full shrink-0 border-b border-mountain-grey pb-4 md:w-48 md:border-b-0 md:border-r md:min-h-[60vh] md:pb-0 md:pr-6">
        <h2 className="mb-4 text-xl font-bold text-ink tracking-widest md:mb-8">{t('admin.title')}</h2>
        <nav className="flex gap-4 overflow-x-auto whitespace-nowrap pb-1 text-sm tracking-widest text-ink-light md:flex-col md:gap-0 md:space-y-4 md:overflow-visible md:whitespace-normal md:pb-0">
          <PreloadNavLink
            to="/admin/dashboard"
            preload={routeLoaders.adminDashboard}
            className={({ isActive }) => isActive ? 'text-ochre font-bold' : 'hover:text-ochre transition-colors'}
          >
            {t('admin.dashboard')}
          </PreloadNavLink>
          <PreloadNavLink
            to="/admin/articles"
            preload={routeLoaders.adminArticles}
            className={({ isActive }) => isActive ? 'text-ochre font-bold' : 'hover:text-ochre transition-colors'}
          >
            {t('admin.articles')}
          </PreloadNavLink>
          <PreloadNavLink
            to="/admin/categories"
            preload={routeLoaders.adminCategories}
            className={({ isActive }) => isActive ? 'text-ochre font-bold' : 'hover:text-ochre transition-colors'}
          >
            {t('admin.categories')}
          </PreloadNavLink>
          <PreloadNavLink
            to="/admin/tags"
            preload={routeLoaders.adminTags}
            className={({ isActive }) => isActive ? 'text-ochre font-bold' : 'hover:text-ochre transition-colors'}
          >
            {t('admin.tags')}
          </PreloadNavLink>
          <div className="hidden h-4 md:block"></div>
          {user?.role === 'admin' && (
            <>
              <PreloadNavLink
                to="/admin/users"
                preload={routeLoaders.adminUsers}
                className={({ isActive }) => isActive ? 'text-ochre font-bold' : 'hover:text-ochre transition-colors'}
              >
                {t('admin.users')}
              </PreloadNavLink>
              <PreloadNavLink
                to="/admin/logs"
                preload={routeLoaders.adminLogs}
                className={({ isActive }) => isActive ? 'text-ochre font-bold' : 'hover:text-ochre transition-colors'}
              >
                {t('admin.logs')}
              </PreloadNavLink>
              <PreloadNavLink
                to="/admin/settings"
                preload={routeLoaders.adminSettings}
                className={({ isActive }) => isActive ? 'text-ochre font-bold' : 'hover:text-ochre transition-colors'}
              >
                {t('admin.settings')}
              </PreloadNavLink>
              <PreloadNavLink
                to="/admin/ai-settings"
                preload={routeLoaders.adminAISettings}
                className={({ isActive }) => isActive ? 'text-ochre font-bold' : 'hover:text-ochre transition-colors'}
              >
                {language === 'zh' ? 'AI 配置' : 'AI Settings'}
              </PreloadNavLink>
              {resumePageEnabled && (
                <PreloadNavLink
                  to="/admin/resume"
                  preload={routeLoaders.adminResumeContent}
                  className={({ isActive }) => isActive ? 'text-ochre font-bold' : 'hover:text-ochre transition-colors'}
                >
                  {t('admin.resume')}
                </PreloadNavLink>
              )}
              {projectsPageEnabled && (
                <PreloadNavLink
                  to="/admin/projects"
                  preload={routeLoaders.adminProjectsContent}
                  className={({ isActive }) => isActive ? 'text-ochre font-bold' : 'hover:text-ochre transition-colors'}
                >
                  {t('admin.projects')}
                </PreloadNavLink>
              )}
            </>
          )}
        </nav>
      </aside>

      <div className="flex-grow">
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
