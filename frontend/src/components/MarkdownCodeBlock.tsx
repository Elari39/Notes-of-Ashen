import { useEffect, useRef, useState } from 'react';
import type { CSSProperties } from 'react';
import SyntaxHighlighter from 'react-syntax-highlighter/dist/esm/prism-light';

const codeFontFamily = '"SFMono-Regular", Consolas, "Liberation Mono", Menlo, monospace';

type MarkdownCodeBlockProps = {
  code: string;
  language: string;
  syntaxTheme: Record<string, CSSProperties>;
};

const MarkdownCodeBlock: React.FC<MarkdownCodeBlockProps> = ({ code, language, syntaxTheme }) => {
  const [copied, setCopied] = useState(false);
  const timerRef = useRef<number>();
  const displayLanguage = language === 'text' ? 'plain text' : language;

  useEffect(() => () => {
    if (timerRef.current) {
      window.clearTimeout(timerRef.current);
    }
  }, []);

  const copyCode = async () => {
    try {
      await navigator.clipboard.writeText(code);
      setCopied(true);
      if (timerRef.current) {
        window.clearTimeout(timerRef.current);
      }
      timerRef.current = window.setTimeout(() => setCopied(false), 1600);
    } catch {
      setCopied(false);
    }
  };

  return (
    <div className="article-code-shell">
      <div className="article-code-toolbar">
        <div className="article-code-dots" aria-hidden="true">
          <span className="bg-[#ef6a5e]" />
          <span className="bg-[#f5bf4f]" />
          <span className="bg-[#63c554]" />
        </div>
        <span className="article-code-language">{displayLanguage}</span>
        <button
          type="button"
          className="article-code-copy"
          onClick={copyCode}
          aria-live="polite"
        >
          {copied ? '已复制' : '复制'}
        </button>
      </div>
      <SyntaxHighlighter
        style={syntaxTheme}
        language={language}
        PreTag="div"
        className="article-code-block"
        codeTagProps={{
          style: {
            color: 'var(--code-ink)',
            fontFamily: codeFontFamily,
            fontWeight: 400,
          },
        }}
        customStyle={{
          margin: 0,
          padding: '1.2rem 1.35rem',
          background: 'var(--code-bg)',
          color: 'var(--code-ink)',
          fontSize: '0.92rem',
          lineHeight: 1.8,
          overflowX: 'auto',
        }}
      >
        {code}
      </SyntaxHighlighter>
    </div>
  );
};

export default MarkdownCodeBlock;
