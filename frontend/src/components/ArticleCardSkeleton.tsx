import React from 'react';
import Skeleton, { SkeletonLine, SkeletonText } from './Skeleton';

export type ArticleCardSkeletonVariant = 'horizontal' | 'standard' | 'alternating';

type ArticleCardSkeletonProps = {
  className?: string;
  variant?: ArticleCardSkeletonVariant;
};

/** 匹配 Home/Search 列表项布局的骨架卡片 */
const ArticleCardSkeleton: React.FC<ArticleCardSkeletonProps> = ({
  className = '',
  variant = 'horizontal',
}) => {
  if (variant === 'standard') {
    return (
      <div className={`overflow-hidden rounded-lg bg-surface-card ${className}`.trim()}>
        <Skeleton className="aspect-[16/9] w-full" />
        <div className="space-y-5 p-6 md:p-8">
          <SkeletonLine width="34%" className="h-3" />
          <SkeletonLine width="78%" className="h-8" />
          <SkeletonText lines={3} />
          <div className="flex gap-4 pt-2">
            <SkeletonLine width="6rem" className="h-3" />
            <SkeletonLine width="5rem" className="h-3" />
          </div>
        </div>
      </div>
    );
  }

  if (variant === 'alternating') {
    return (
      <div className={`flex flex-col overflow-hidden rounded-lg bg-surface-card md:min-h-64 md:flex-row ${className}`.trim()}>
        <Skeleton className="aspect-[16/9] w-full shrink-0 md:h-auto md:w-[42%] md:aspect-auto" />
        <div className="flex flex-1 flex-col justify-between gap-6 p-6 md:p-8">
          <div className="space-y-5">
            <SkeletonLine width="32%" className="h-3" />
            <SkeletonLine width="76%" className="h-8" />
            <SkeletonText lines={3} />
          </div>
          <div className="flex gap-4">
            <SkeletonLine width="6rem" className="h-3" />
            <SkeletonLine width="5rem" className="h-3" />
          </div>
        </div>
      </div>
    );
  }

  return (
    <div className={`group relative flex flex-col gap-5 pb-2 items-start md:flex-row md:gap-8 md:pb-0 ${className}`.trim()}>
      <Skeleton className="aspect-[16/9] w-full shrink-0 md:h-48 md:w-1/3" />
      <div className="flex-1 space-y-5">
        <SkeletonLine width="70%" className="h-7" />
        <SkeletonText lines={3} />
        <div className="flex gap-4">
          <SkeletonLine width="6rem" className="h-3" />
          <SkeletonLine width="5rem" className="h-3" />
        </div>
      </div>
    </div>
  );
};

export default ArticleCardSkeleton;
