import { useState, type ReactNode } from 'react';
import { usePreferenceStore } from '../store/preferences';
import { translate } from '../i18n';

type MarkdownTableProps = {
  children: ReactNode;
};

const MarkdownTable = ({ children }: MarkdownTableProps) => {
  const [collapsed, setCollapsed] = useState(false);
  // 订阅 preference store，使 i18n 切换时工具栏文案立即重渲。
  const language = usePreferenceStore((state) => state.language);
  const t = (key: Parameters<typeof translate>[1]) => translate(language, key);

  return (
    <div className="article-table-shell">
      <div className="article-table-toolbar">
        <span>{t('markdownTable.table')}</span>
        <button
          type="button"
          onClick={() => setCollapsed((value) => !value)}
          aria-expanded={!collapsed}
        >
          {collapsed ? t('markdownTable.expand') : t('markdownTable.collapse')}
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
