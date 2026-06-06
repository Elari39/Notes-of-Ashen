import React from 'react';

interface PaginationProps {
  currentPage: number;
  total: number;
  pageSize: number;
  onPageChange: (page: number) => void;
}

const Pagination: React.FC<PaginationProps> = ({ currentPage, total, pageSize, onPageChange }) => {
  const totalPages = Math.ceil(total / pageSize);

  if (totalPages <= 1) return null;

  return (
    <div className="flex justify-center items-center space-x-6 mt-16 pt-8 border-t border-mountain-grey border-opacity-50">
      <button
        onClick={() => onPageChange(currentPage - 1)}
        disabled={currentPage === 1}
        className="text-ink-light hover:text-ochre disabled:opacity-30 disabled:hover:text-ink-light transition-colors tracking-widest text-sm"
      >
        上一卷
      </button>
      
      <span className="text-sm text-ink font-serif">
        第 {currentPage} / {totalPages} 卷
      </span>

      <button
        onClick={() => onPageChange(currentPage + 1)}
        disabled={currentPage === totalPages}
        className="text-ink-light hover:text-ochre disabled:opacity-30 disabled:hover:text-ink-light transition-colors tracking-widest text-sm"
      >
        下一卷
      </button>
    </div>
  );
};

export default Pagination;
