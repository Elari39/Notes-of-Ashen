import React from 'react';
import Skeleton from '../Skeleton';

type TableSkeletonProps = {
  rows?: number;
  cols?: number;
  className?: string;
};

/**
 * 表格骨架屏：admin 列表加载期占位，保证视觉不闪。
 * 与 admin-responsive-table 移动端卡片化布局兼容（小屏时改为多行块）。
 */
const TableSkeleton: React.FC<TableSkeletonProps> = ({ rows = 5, cols = 4, className = '' }) => {
  return (
    <div className={['space-y-2', className].filter(Boolean).join(' ')} aria-hidden="true">
      {/* 表头骨架 */}
      <div
        className="hidden gap-4 border-b border-mountain-grey px-2 py-3 md:grid"
        style={{ gridTemplateColumns: `repeat(${cols}, minmax(0, 1fr))` }}
      >
        {Array.from({ length: cols }).map((_, i) => (
          <Skeleton key={i} className="h-3 w-20" />
        ))}
      </div>
      {Array.from({ length: rows }).map((_, rowIdx) => (
        <div
          key={rowIdx}
          className="grid grid-cols-1 gap-3 border border-mountain-grey p-3 md:grid-cols-[repeat(var(--cols),minmax(0,1fr))] md:gap-4 md:border-0 md:border-b md:p-2"
          style={{ ['--cols' as never]: cols }}
        >
          {Array.from({ length: cols }).map((_, colIdx) => (
            <Skeleton
              key={colIdx}
              className="h-4"
              style={{ width: `${60 + ((rowIdx + colIdx) % 4) * 10}%` }}
            />
          ))}
        </div>
      ))}
    </div>
  );
};

export default TableSkeleton;
