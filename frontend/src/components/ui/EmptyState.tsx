import React from 'react';

type EmptyStateIllustration = 'ink-drop' | 'leaf' | 'cloud';

export type EmptyStateProps = {
  /** 内嵌 SVG 类型，自动跟随 currentColor（≤ 400B 每个） */
  illustration?: EmptyStateIllustration;
  title: string;
  description?: string;
  action?: { label: string; onClick(): void } | React.ReactNode;
  className?: string;
};

type ActionObject = { label: string; onClick(): void };

const isActionObject = (value: unknown): value is ActionObject =>
  typeof value === 'object' &&
  value !== null &&
  'label' in value &&
  typeof (value as { label: unknown }).label === 'string' &&
  'onClick' in value &&
  typeof (value as { onClick: unknown }).onClick === 'function';

const Illustrations: Record<EmptyStateIllustration, React.ReactNode> = {
  // 墨滴：笔触感圆形，54×54
  'ink-drop': (
    <svg width="54" height="54" viewBox="0 0 54 54" fill="none" aria-hidden="true">
      <path
        d="M27 4C27 4 14 24 14 34C14 42.28 20.72 50 27 50C33.28 50 40 42.28 40 34C40 24 27 4 27 4Z"
        fill="currentColor"
        fillOpacity="0.12"
        stroke="currentColor"
        strokeWidth="1.5"
        strokeLinecap="round"
        strokeLinejoin="round"
      />
      <circle cx="25" cy="32" r="2.5" fill="currentColor" fillOpacity="0.28" />
    </svg>
  ),
  // 叶片：54×54，两片叶
  'leaf': (
    <svg width="54" height="54" viewBox="0 0 54 54" fill="none" aria-hidden="true">
      <path
        d="M27 6C27 6 8 20 8 34C8 38 12 44 18 44C24 44 27 36 27 36C27 36 30 44 36 44C42 44 46 38 46 34C46 20 27 6 27 6Z"
        fill="currentColor"
        fillOpacity="0.12"
        stroke="currentColor"
        strokeWidth="1.5"
        strokeLinecap="round"
        strokeLinejoin="round"
      />
      <path
        d="M27 26V36"
        stroke="currentColor"
        strokeWidth="1.5"
        strokeLinecap="round"
        opacity="0.4"
      />
    </svg>
  ),
  // 云：54×54
  'cloud': (
    <svg width="54" height="54" viewBox="0 0 54 54" fill="none" aria-hidden="true">
      <path
        d="M14 38C10.7 38 8 35.3 8 32C8 28.7 10.7 26 14 26H14.5C14.2 25.1 14 24.1 14 23C14 17.5 18.5 13 24 13C28.5 13 32.3 16 33.5 20H34C38.4 20 42 23.6 42 28C42 32.4 38.4 36 34 36H14Z"
        fill="currentColor"
        fillOpacity="0.08"
        stroke="currentColor"
        strokeWidth="1.5"
        strokeLinecap="round"
        strokeLinejoin="round"
      />
      <line
        x1="18" y1="40" x2="36" y2="40"
        stroke="currentColor"
        strokeWidth="1.5"
        strokeLinecap="round"
        opacity="0.5"
      />
      <line
        x1="22" y1="44" x2="32" y2="44"
        stroke="currentColor"
        strokeWidth="1.5"
        strokeLinecap="round"
        opacity="0.3"
      />
    </svg>
  ),
};

/**
 * 空状态占位：SVG 插画 + 标题 + 可选描述 + 行动入口。
 * 自动跟随 currentColor，暗色不变。
 */
const EmptyState: React.FC<EmptyStateProps> = ({
  illustration,
  title,
  description,
  action,
  className = '',
}) => {
  return (
    <div
      className={[
        'flex flex-col items-center justify-center gap-4 py-12 px-4 text-center',
        className,
      ]
        .filter(Boolean)
        .join(' ')}
    >
      {illustration && (
        <div className="text-ink-light opacity-50">{Illustrations[illustration]}</div>
      )}
      <p className="text-sm tracking-widest text-ink-light">{title}</p>
      {description && (
        <p className="max-w-xs text-xs leading-relaxed text-ink-light opacity-70">
          {description}
        </p>
      )}
      {action && (
        <div className="pt-1">
          {isActionObject(action) ? (
            <button
              type="button"
              onClick={action.onClick}
              className="inline-flex items-center gap-2 border border-mountain-grey px-4 py-1.5 text-xs tracking-widest text-ink-light transition-colors duration-fast hover:border-ochre hover:text-ochre focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-ochre"
            >
              {action.label}
            </button>
          ) : (
            action
          )}
        </div>
      )}
    </div>
  );
};

export default EmptyState;