import React, { useState } from 'react';
import { formatText, translate } from '../i18n';
import { usePreferenceStore } from '../store/preferences';

interface PaginationProps {
  currentPage: number;
  total: number;
  pageSize: number;
  onPageChange: (page: number) => void;
  /** 当 totalPages > 7 时显示页码块 */
  showPageNumbers?: boolean;
  /** 显示跳转输入框 */
  withJump?: boolean;
}

/** 生成省略号页码序列：始终带 1 和 last，中间显示当前页 ±2 */
const buildPageList = (current: number, total: number): (number | 'gap')[] => {
  if (total <= 7) {
    return Array.from({ length: total }, (_, i) => i + 1);
  }
  const items: (number | 'gap')[] = [];
  const window = 1;
  const start = Math.max(2, current - window);
  const end = Math.min(total - 1, current + window);
  items.push(1);
  if (start > 2) items.push('gap');
  for (let i = start; i <= end; i++) items.push(i);
  if (end < total - 1) items.push('gap');
  items.push(total);
  return items;
};

const Pagination: React.FC<PaginationProps> = ({
  currentPage,
  total,
  pageSize,
  onPageChange,
  showPageNumbers = false,
  withJump = false,
}) => {
  const language = usePreferenceStore((state) => state.language);
  const totalPages = Math.ceil(total / pageSize);
  const t = (key: Parameters<typeof translate>[1]) => translate(language, key);
  const [jumpValue, setJumpValue] = useState('');

  if (totalPages <= 1) return null;

  const enablePageList = showPageNumbers && totalPages > 1;

  const handleJump = (e: React.FormEvent) => {
    e.preventDefault();
    const n = Number.parseInt(jumpValue, 10);
    if (!Number.isFinite(n) || n < 1 || n > totalPages) return;
    onPageChange(n);
    setJumpValue('');
  };

  const itemClass = 'min-h-11 min-w-11 rounded-md border border-hairline px-3 py-2 text-sm text-muted hover:border-ink hover:text-ink transition-colors duration-fast focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-ochre disabled:opacity-30 disabled:hover:text-muted';

  return (
    <div className="flex flex-wrap justify-center items-center gap-x-4 gap-y-2 mt-16 pt-8 border-t border-mountain-grey/50">
      <button
        type="button"
        onClick={() => onPageChange(currentPage - 1)}
        disabled={currentPage === 1}
        aria-disabled={currentPage === 1}
        className={itemClass}
      >
        {t('pagination.prev')}
      </button>

      {enablePageList ? (
        <ul className="flex items-center gap-1">
          {buildPageList(currentPage, totalPages).map((it, idx) =>
            it === 'gap' ? (
              <li key={`gap-${idx}`} className="px-1 text-ink-light opacity-50 select-none">…</li>
            ) : (
              <li key={it}>
                <button
                  type="button"
                  onClick={() => onPageChange(it)}
                  aria-current={it === currentPage ? 'page' : undefined}
                  className={[
                    'min-h-11 min-w-11 rounded-md px-2 py-2 text-sm transition-colors duration-fast focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-ochre',
                    it === currentPage
                      ? 'bg-ochre text-on-accent'
                      : 'text-muted hover:bg-surface-soft hover:text-ink',
                  ].join(' ')}
                >
                  {it}
                </button>
              </li>
            ),
          )}
        </ul>
      ) : (
        <span className="font-display text-xl text-ink" aria-current="page">
          {formatText(t('pagination.page'), { current: currentPage, total: totalPages })}
        </span>
      )}

      <button
        type="button"
        onClick={() => onPageChange(currentPage + 1)}
        disabled={currentPage === totalPages}
        aria-disabled={currentPage === totalPages}
        className={itemClass}
      >
        {t('pagination.next')}
      </button>

      {withJump && totalPages > 7 && (
        <form onSubmit={handleJump} className="ml-2 flex items-center gap-2 text-xs text-ink-light tracking-widest">
          <label className="opacity-70" htmlFor="pagination-jump-input">
            {t('pagination.jumpLabel')}
          </label>
          <input
            id="pagination-jump-input"
            type="text"
            inputMode="numeric"
            value={jumpValue}
            onChange={(e) => setJumpValue(e.target.value.replace(/[^0-9]/g, ''))}
            aria-label={t('pagination.jumpLabel')}
            className="w-14 border-b border-mountain-grey bg-transparent py-1 px-1 text-center text-sm text-ink focus:outline-hidden focus:border-ochre transition-colors duration-fast"
          />
          <button
            type="submit"
            className="border border-mountain-grey px-2 py-1 hover:border-ochre hover:text-ochre transition-colors duration-fast focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-ochre"
          >
            {t('pagination.jumpGo')}
          </button>
        </form>
      )}
    </div>
  );
};

export default Pagination;
