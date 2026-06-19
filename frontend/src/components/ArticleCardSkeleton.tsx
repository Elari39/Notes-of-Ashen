import React from 'react';
import Skeleton, { SkeletonLine, SkeletonText } from './Skeleton';

/** 匹配 Home/Search 列表项布局的骨架卡片 */
const ArticleCardSkeleton: React.FC<{ className?: string }> = ({ className = '' }) => (
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

export default ArticleCardSkeleton;
