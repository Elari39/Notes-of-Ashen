import React from 'react';
import { Outlet, NavLink } from 'react-router-dom';

const AdminLayout: React.FC = () => {
  return (
    <div className="flex flex-col md:flex-row gap-8 max-w-7xl mx-auto w-full">
      {/* Admin Sidebar */}
      <aside className="w-full md:w-48 shrink-0 border-r border-mountain-grey md:min-h-[60vh] pr-6">
        <h2 className="text-xl font-bold text-ink mb-8 tracking-widest">掌卷</h2>
        <nav className="flex flex-col space-y-4 text-ink-light tracking-widest text-sm">
          <NavLink 
            to="/admin/articles" 
            className={({isActive}) => isActive ? "text-ochre font-bold" : "hover:text-ochre transition-colors"}
          >
            文章管理
          </NavLink>
          <NavLink 
            to="/admin/categories" 
            className={({isActive}) => isActive ? "text-ochre font-bold" : "hover:text-ochre transition-colors"}
          >
            分类管理
          </NavLink>
          <NavLink 
            to="/admin/tags" 
            className={({isActive}) => isActive ? "text-ochre font-bold" : "hover:text-ochre transition-colors"}
          >
            标签管理
          </NavLink>
          <div className="h-4"></div> {/* spacer */}
          <NavLink 
            to="/admin/users" 
            className={({isActive}) => isActive ? "text-ochre font-bold" : "hover:text-ochre transition-colors"}
          >
            掌印 (用户)
          </NavLink>
          <NavLink 
            to="/admin/logs" 
            className={({isActive}) => isActive ? "text-ochre font-bold" : "hover:text-ochre transition-colors"}
          >
            青史 (日志)
          </NavLink>
        </nav>
      </aside>
      
      {/* Admin Content */}
      <div className="flex-grow">
        <Outlet />
      </div>
    </div>
  );
};

export default AdminLayout;
