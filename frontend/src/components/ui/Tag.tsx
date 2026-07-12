import React from 'react';
import { usePreferenceStore } from '../../store/preferences';
import { translate } from '../../i18n';

type TagTone = 'neutral' | 'ochre' | 'success' | 'warning' | 'danger' | 'info';
type Size = 'sm' | 'md';

export type TagProps = {
  tone?: TagTone;
  size?: Size;
  interactive?: boolean;
  onRemove?: () => void;
  children: React.ReactNode;
  className?: string;
};

const toneClass: Record<TagTone, string> = {
  neutral: 'border-transparent text-ink bg-surface-card',
  ochre: 'border-[var(--code-border)] text-ochre bg-[var(--inline-code-bg)]',
  success: 'border-[var(--moss-soft)] text-moss bg-[var(--moss-soft)]',
  warning: 'border-[var(--amber-soft)] text-amber bg-[var(--amber-soft)]',
  danger: 'border-[var(--ember-soft)] text-ember bg-[var(--ember-soft)]',
  info: 'border-[var(--dusk-soft)] text-dusk bg-[var(--dusk-soft)]',
};

const sizeClass: Record<Size, string> = {
  sm: 'px-2 py-0.5 text-[0.7rem] leading-relaxed',
  md: 'px-2.5 py-0.5 text-xs leading-relaxed',
};

const baseClass =
  'inline-flex items-center gap-1 rounded-full border font-medium transition-colors duration-fast ease-paper';
const interactiveClass =
  'cursor-pointer hover:border-ink-light hover:text-ink focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-1 focus-visible:outline-ochre';

/**
 * 状态标签：用于文章状态、角色徽章、分类/标签云、管理页状态列。
 */
const Tag: React.FC<TagProps> = ({
  tone = 'neutral',
  size = 'sm',
  interactive = false,
  onRemove,
  children,
  className = '',
}) => {
  const language = usePreferenceStore((state) => state.language);
  return (
    <span
      className={[
        baseClass,
        toneClass[tone],
        sizeClass[size],
        interactive ? interactiveClass : '',
        className,
      ]
        .filter(Boolean)
        .join(' ')}
    >
      {children}
      {onRemove && (
        <button
          type="button"
          onClick={(e) => {
            e.stopPropagation();
            onRemove();
          }}
          aria-label={translate(language, 'common.remove')}
          className="inline-flex items-center justify-center min-w-[44px] min-h-[44px] -my-2 p-0.5 opacity-60 hover:opacity-100 transition-opacity"
        >
          ✕
        </button>
      )}
    </span>
  );
};

export default Tag;
