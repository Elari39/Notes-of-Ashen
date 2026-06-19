import React from 'react';
import { formatText, translate } from '../i18n';
import { usePreferenceStore } from '../store/preferences';

interface PaginationProps {
  currentPage: number;
  total: number;
  pageSize: number;
  onPageChange: (page: number) => void;
}

const Pagination: React.FC<PaginationProps> = ({ currentPage, total, pageSize, onPageChange }) => {
  const language = usePreferenceStore((state) => state.language);
  const totalPages = Math.ceil(total / pageSize);
  const t = (key: Parameters<typeof translate>[1]) => translate(language, key);

  if (totalPages <= 1) return null;

  return (
    <div className="flex justify-center items-center space-x-6 mt-16 pt-8 border-t border-mountain-grey border-opacity-50">
      <button
        type="button"
        onClick={() => onPageChange(currentPage - 1)}
        disabled={currentPage === 1}
        aria-disabled={currentPage === 1}
        className="text-ink-light hover:text-ochre disabled:opacity-30 disabled:hover:text-ink-light transition-colors tracking-widest text-sm"
      >
        {t('pagination.prev')}
      </button>

      <span className="text-sm text-ink font-serif" aria-current="page">
        {formatText(t('pagination.page'), { current: currentPage, total: totalPages })}
      </span>

      <button
        type="button"
        onClick={() => onPageChange(currentPage + 1)}
        disabled={currentPage === totalPages}
        aria-disabled={currentPage === totalPages}
        className="text-ink-light hover:text-ochre disabled:opacity-30 disabled:hover:text-ink-light transition-colors tracking-widest text-sm"
      >
        {t('pagination.next')}
      </button>
    </div>
  );
};

export default Pagination;
