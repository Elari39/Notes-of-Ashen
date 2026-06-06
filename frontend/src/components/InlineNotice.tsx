import React from 'react';

type InlineNoticeProps = {
  message?: string;
  tone?: 'error' | 'success';
  className?: string;
};

const InlineNotice: React.FC<InlineNoticeProps> = ({ message, tone = 'error', className = '' }) => {
  if (!message) return null;

  const toneClass = tone === 'success'
    ? 'border-ink-light text-ink-light'
    : 'border-ochre text-ochre';

  return (
    <p
      role={tone === 'error' ? 'alert' : 'status'}
      className={`text-sm leading-relaxed px-4 py-3 border-l-2 bg-[var(--notice-bg)] ${toneClass} ${className}`}
    >
      {message}
    </p>
  );
};

export default InlineNotice;
