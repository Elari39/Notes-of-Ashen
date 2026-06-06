import React from 'react';
import { Outlet, NavLink } from 'react-router-dom';
import { translate } from '../../i18n';
import { usePreferenceStore } from '../../store/preferences';

const AdminLayout: React.FC = () => {
  const language = usePreferenceStore((state) => state.language);
  const t = (key: Parameters<typeof translate>[1]) => translate(language, key);

  return (
    <div className="flex flex-col md:flex-row gap-8 max-w-7xl mx-auto w-full">
      <aside className="w-full md:w-48 shrink-0 border-r border-mountain-grey md:min-h-[60vh] pr-6">
        <h2 className="text-xl font-bold text-ink mb-8 tracking-widest">{t('admin.title')}</h2>
        <nav className="flex flex-col space-y-4 text-ink-light tracking-widest text-sm">
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
        </nav>
      </aside>

      <div className="flex-grow">
        <Outlet />
      </div>
    </div>
  );
};

export default AdminLayout;
