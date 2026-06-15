import { useState, type ReactNode } from 'react';

const markdownTableLabels = () => {
  const isEnglish = typeof localStorage !== 'undefined' && localStorage.getItem('notesOfAshen.language') === 'en'
    || typeof document !== 'undefined' && document.documentElement.lang.toLowerCase().startsWith('en');
  return isEnglish
    ? { table: 'Table', collapse: 'Collapse', expand: 'Expand' }
    : { table: '表格', collapse: '收起', expand: '展开' };
};

type MarkdownTableProps = {
  children: ReactNode;
};

const MarkdownTable = ({ children }: MarkdownTableProps) => {
  const [collapsed, setCollapsed] = useState(false);
  const labels = markdownTableLabels();

  return (
    <div className="article-table-shell">
      <div className="article-table-toolbar">
        <span>{labels.table}</span>
        <button
          type="button"
          onClick={() => setCollapsed((value) => !value)}
          aria-expanded={!collapsed}
        >
          {collapsed ? labels.expand : labels.collapse}
        </button>
      </div>
      {!collapsed && (
        <div className="article-table-wrap">
          <table>{children}</table>
        </div>
      )}
    </div>
  );
};

export default MarkdownTable;
