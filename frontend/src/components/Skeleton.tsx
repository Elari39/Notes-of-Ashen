import React from 'react';

type SkeletonProps = {
  className?: string;
  style?: React.CSSProperties;
};

/** 基础骨架块，复用 --paper-soft / --mounten-grey 变量，pulse 动画见 index.css .skeleton */
const Skeleton: React.FC<SkeletonProps> = ({ className = '', style }) => (
  <div className={`skeleton ${className}`.trim()} aria-hidden="true" style={style} />
);

/** 单行文字占位 */
export const SkeletonLine: React.FC<{ width?: string; className?: string }> = ({
  width = '100%',
  className = '',
}) => <Skeleton className={`h-3.5 ${className}`.trim()} style={{ width }} />;

/** 多行段落占位 */
export const SkeletonText: React.FC<{ lines?: number; className?: string }> = ({
  lines = 3,
  className = '',
}) => (
  <div className={`space-y-2.5 ${className}`.trim()}>
    {Array.from({ length: lines }).map((_, index) => (
      <SkeletonLine
        key={index}
        width={index === lines - 1 ? '60%' : '100%'}
      />
    ))}
  </div>
);

export default Skeleton;
