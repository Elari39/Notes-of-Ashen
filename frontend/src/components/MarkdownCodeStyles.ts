import type { CSSProperties } from 'react';

export const codeFontFamily = '"JetBrains Mono", "SFMono-Regular", Consolas, "Liberation Mono", Menlo, monospace';

export const codeBlockStyle: CSSProperties = {
  margin: 0,
  padding: '1.25rem 1.4rem',
  background: 'var(--code-bg)',
  color: 'var(--code-ink)',
  fontSize: '0.92rem',
  lineHeight: 1.72,
  overflowX: 'auto',
};

export const codeTagStyle: CSSProperties = {
  color: 'var(--code-ink)',
  fontFamily: codeFontFamily,
  fontWeight: 400,
};
