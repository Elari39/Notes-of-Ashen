import { useState, type ReactNode } from 'react';
import { usePreferenceStore } from '../store/preferences';

type MarkdownTableLabels = {
  table: string;
  collapse: string;
  expand: string;
};

const markdownTableLabels = (language: 'zh' | 'en'): MarkdownTableLabels => {
  return language === 'en'
    ? { table: 'Table', collapse: 'Collapse', expand: 'Expand' }
    : { table: '表格', collapse: '收起', expand: '展开' };
};

type MarkdownTableProps = {
  children: ReactNode;
};

const MarkdownTable = ({ children }: MarkdownTableProps) => {
  const [collapsed, setCollapsed] = useState(false);
  // 订阅 preference store，使 i18n 切换时工具栏文案立即重渲。
  const language = usePreferenceStore((state) => state.language);
  const labels = markdownTableLabels(language);

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
