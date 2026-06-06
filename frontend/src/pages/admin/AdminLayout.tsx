import React from 'react';
import { NavLink, useLocation, useOutlet } from 'react-router-dom';
import { AnimatePresence, motion, useReducedMotion } from 'framer-motion';
import { translate } from '../../i18n';
import { usePreferenceStore } from '../../store/preferences';
import { useAuthStore } from '../../store/auth';

const AdminLayout: React.FC = () => {
  const language = usePreferenceStore((state) => state.language);
  const user = useAuthStore((state) => state.user);
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
    <div className="flex flex-col md:flex-row gap-8 max-w-7xl mx-auto w-full">
      <aside className="w-full md:w-48 shrink-0 border-r border-mountain-grey md:min-h-[60vh] pr-6">
        <h2 className="text-xl font-bold text-ink mb-8 tracking-widest">{t('admin.title')}</h2>
        <nav className="flex flex-col space-y-4 text-ink-light tracking-widest text-sm">
          <NavLink
            to="/admin/dashboard"
            className={({ isActive }) => isActive ? 'text-ochre font-bold' : 'hover:text-ochre transition-colors'}
          >
            Dashboard
          </NavLink>
          <NavLink
            to="/admin/articles"
            className={({ isActive }) => isActive ? 'text-ochre font-bold' : 'hover:text-ochre transition-colors'}
          >
            {t('admin.articles')}
          </NavLink>
          <NavLink
            to="/admin/categories"
            className={({ isActive }) => isActive ? 'text-ochre font-bold' : 'hover:text-ochre transition-colors'}
          >
            {t('admin.categories')}
          </NavLink>
          <NavLink
            to="/admin/tags"
            className={({ isActive }) => isActive ? 'text-ochre font-bold' : 'hover:text-ochre transition-colors'}
          >
            {t('admin.tags')}
          </NavLink>
          <div className="h-4"></div>
          {user?.role === 'admin' && (
            <>
              <NavLink
                to="/admin/users"
                className={({ isActive }) => isActive ? 'text-ochre font-bold' : 'hover:text-ochre transition-colors'}
              >
                {t('admin.users')}
              </NavLink>
              <NavLink
                to="/admin/logs"
                className={({ isActive }) => isActive ? 'text-ochre font-bold' : 'hover:text-ochre transition-colors'}
              >
                {t('admin.logs')}
              </NavLink>
              <NavLink
                to="/admin/settings"
                className={({ isActive }) => isActive ? 'text-ochre font-bold' : 'hover:text-ochre transition-colors'}
              >
                {t('admin.settings')}
              </NavLink>
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
