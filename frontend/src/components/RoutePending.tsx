import React from 'react';
import { usePreferenceStore } from '../store/preferences';

type PendingVariant = 'page' | 'admin' | 'inline';

type PagePendingStateProps = {
  label?: string;
  variant?: PendingVariant;
  className?: string;
};

const pendingLabels = {
  zh: {
    page: '页面加载中',
    admin: '内容加载中',
    inline: '正在更新',
  },
  en: {
    page: 'Loading page',
    admin: 'Loading content',
    inline: 'Updating',
  },
} as const;

export const RoutePendingIndicator: React.FC<{ className?: string }> = ({ className = '' }) => (
  <div className={`route-ember-indicator ${className}`.trim()} aria-hidden="true">
    <span />
  </div>
);

const PagePendingState: React.FC<PagePendingStateProps> = ({
  label,
  variant = 'page',
  className = '',
}) => {
  const language = usePreferenceStore((state) => state.language);
  const fallbackLabel = pendingLabels[language][variant];
  const isInline = variant === 'inline';
  const containerClass = isInline
    ? 'mb-5 flex items-center gap-3 border border-mountain-grey bg-[var(--paper-soft)] px-4 py-3 text-sm tracking-widest text-ink-light'
    : 'flex min-h-48 flex-col items-center justify-center gap-4 px-6 py-14 text-center text-sm tracking-[0.2em] text-ink-light';

  return (
    <div
      role="status"
      aria-live="polite"
      className={`${containerClass} ${className}`.trim()}
    >
      <span className="ember-dot-row" aria-hidden="true">
        <span />
        <span />
        <span />
      </span>
      <span>{label || fallbackLabel}</span>
    </div>
  );
};

export default PagePendingState;
