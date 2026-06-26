import React from 'react';
import { usePreferenceStore } from '../store/preferences';
import { translate } from '../i18n';

export type InlineNoticeTone = 'error' | 'success' | 'warning' | 'info';

type InlineNoticeProps = {
  message?: string;
  tone?: InlineNoticeTone;
  /** 是否显示默认图标；可传 ReactNode 自定义 */
  icon?: boolean | React.ReactNode;
  onDismiss?: () => void;
  /** 行动按钮 slot（例如"重试"） */
  action?: React.ReactNode;
  className?: string;
};

const toneClass: Record<InlineNoticeTone, string> = {
  error: 'border-ember text-ember',
  success: 'border-moss text-moss',
  warning: 'border-amber text-amber',
  info: 'border-dusk text-dusk',
};

const DefaultIcon: React.FC<{ tone: InlineNoticeTone }> = ({ tone }) => {
  // 极简内嵌 SVG（每个 < 200B）
  switch (tone) {
    case 'success':
      return (
        <svg width="14" height="14" viewBox="0 0 14 14" fill="none" aria-hidden="true">
          <path d="M3 7.5L6 10.5L11.5 4" stroke="currentColor" strokeWidth="1.5" strokeLinecap="round" strokeLinejoin="round" />
        </svg>
      );
    case 'warning':
      return (
        <svg width="14" height="14" viewBox="0 0 14 14" fill="none" aria-hidden="true">
          <path d="M7 2L13 12H1L7 2Z" stroke="currentColor" strokeWidth="1.5" strokeLinejoin="round" />
          <path d="M7 6V8" stroke="currentColor" strokeWidth="1.5" strokeLinecap="round" />
          <circle cx="7" cy="10" r="0.6" fill="currentColor" />
        </svg>
      );
    case 'info':
      return (
        <svg width="14" height="14" viewBox="0 0 14 14" fill="none" aria-hidden="true">
          <circle cx="7" cy="7" r="5.5" stroke="currentColor" strokeWidth="1.5" />
          <path d="M7 6.5V10" stroke="currentColor" strokeWidth="1.5" strokeLinecap="round" />
          <circle cx="7" cy="4.2" r="0.6" fill="currentColor" />
        </svg>
      );
    case 'error':
    default:
      return (
        <svg width="14" height="14" viewBox="0 0 14 14" fill="none" aria-hidden="true">
          <circle cx="7" cy="7" r="5.5" stroke="currentColor" strokeWidth="1.5" />
          <path d="M4.5 4.5L9.5 9.5M9.5 4.5L4.5 9.5" stroke="currentColor" strokeWidth="1.5" strokeLinecap="round" />
        </svg>
      );
  }
};

/**
 * 内联通知。
 * - 4 态：error(默认) / success / warning / info
 * - 兼容旧 props：`tone?: 'error' | 'success'` 调用方无需改动
 * - 新增可选 icon / onDismiss / action
 */
const InlineNotice: React.FC<InlineNoticeProps> = ({
  message,
  tone = 'error',
  icon,
  onDismiss,
  action,
  className = '',
}) => {
  const language = usePreferenceStore((state) => state.language);
  if (!message) return null;

  const role = tone === 'error' || tone === 'warning' ? 'alert' : 'status';
  const ariaLive = tone === 'error' || tone === 'warning' ? 'assertive' : 'polite';

  const iconNode =
    icon === false || icon === undefined
      ? null
      : icon === true
      ? <DefaultIcon tone={tone} />
      : icon;

  return (
    <div
      role={role}
      aria-live={ariaLive}
      className={[
        'flex items-start gap-2 border-l-2 bg-[var(--notice-bg)] px-4 py-3 text-sm leading-relaxed',
        toneClass[tone],
        className,
      ]
        .filter(Boolean)
        .join(' ')}
    >
      {iconNode && (
        <span className="shrink-0 inline-flex items-center pt-1" aria-hidden="true">
          {iconNode}
        </span>
      )}
      <span className="flex-1">{message}</span>
      {action && <span className="shrink-0 inline-flex items-center">{action}</span>}
      {onDismiss && (
        <button
          type="button"
          onClick={onDismiss}
          aria-label={translate(language, 'common.dismiss')}
          className="shrink-0 inline-flex items-center justify-center min-w-[44px] min-h-[44px] -my-2 opacity-60 hover:opacity-100 transition-opacity"
        >
          ✕
        </button>
      )}
    </div>
  );
};

export default InlineNotice;
