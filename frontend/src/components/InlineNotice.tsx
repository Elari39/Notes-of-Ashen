import React from 'react';

type InlineNoticeProps = {
  message?: string;
  tone?: 'error' | 'success';
  className?: string;
};

const InlineNotice: React.FC<InlineNoticeProps> = ({ message, tone = 'error', className = '' }) => {
  if (!message) return null;

  const toneClass = tone === 'success'
    ? 'border-ink-light text-ink-light bg-white bg-opacity-30'
    : 'border-ochre text-ochre bg-white bg-opacity-40';

  return (
    <p
      role={tone === 'error' ? 'alert' : 'status'}
      className={`text-sm leading-relaxed px-4 py-3 border-l-2 ${toneClass} ${className}`}
    >
      {message}
    </p>
  );
};

export default InlineNotice;
