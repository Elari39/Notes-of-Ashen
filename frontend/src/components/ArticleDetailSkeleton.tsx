import React from 'react';
import Skeleton, { SkeletonLine, SkeletonText } from './Skeleton';

/** 匹配文章详情页主体布局的骨架 */
const ArticleDetailSkeleton: React.FC<{ className?: string }> = ({ className = '' }) => (
  <div className={`mx-auto w-full max-w-[46rem] ${className}`.trim()}>
    <header className="mb-16 text-center">
      <Skeleton className="mb-12 h-64 w-full md:h-80" />
      <SkeletonLine width="50%" className="mx-auto h-9" />
      <div className="mt-6 flex justify-center gap-4">
        <SkeletonLine width="5rem" className="h-3" />
        <SkeletonLine width="4rem" className="h-3" />
        <SkeletonLine width="4rem" className="h-3" />
      </div>
    </header>
    <div className="space-y-6">
      {Array.from({ length: 5 }).map((_, index) => (
        <SkeletonText key={index} lines={4} />
      ))}
    </div>
  </div>
);

export default ArticleDetailSkeleton;
