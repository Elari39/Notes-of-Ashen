import { useEffect, useRef, useState, type ReactNode } from 'react';
import { translate } from '../i18n';
import { usePreferenceStore } from '../store/preferences';

type MarkdownCodeToolbarProps = {
  code: string;
  language: string;
  actions?: ReactNode;
};

const copyWithLegacyApi = (code: string) => {
  const input = document.createElement('textarea');
  input.value = code;
  input.setAttribute('readonly', '');
  input.style.position = 'fixed';
  input.style.top = '0';
  input.style.left = '-9999px';
  input.style.opacity = '0';
  input.style.pointerEvents = 'none';
  document.body.appendChild(input);
  try {
    input.focus();
    input.select();
    if (!document.execCommand('copy')) {
      throw new Error('copy failed');
    }
  } finally {
    input.remove();
  }
};

const MarkdownCodeToolbar = ({ code, language, actions }: MarkdownCodeToolbarProps) => {
  const [copied, setCopied] = useState(false);
  const timerRef = useRef<number | undefined>(undefined);
  const uiLanguage = usePreferenceStore((state) => state.language);
  const t = (key: Parameters<typeof translate>[1]) => translate(uiLanguage, key);
  const displayLanguage = language === 'text' ? 'plain text' : language;

  useEffect(() => () => {
    if (timerRef.current) {
      window.clearTimeout(timerRef.current);
    }
  }, []);

  const copyCode = async () => {
    try {
      if (navigator.clipboard?.writeText) {
        try {
          await navigator.clipboard.writeText(code);
        } catch {
          // 本地开发环境或非安全上下文可能拒绝异步剪贴板，继续使用兼容路径。
          copyWithLegacyApi(code);
        }
      } else {
        copyWithLegacyApi(code);
      }
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
    <div className="article-code-toolbar">
      <div className="article-code-dots" aria-hidden="true">
        <span className="bg-[#ef6a5e]" />
        <span className="bg-[#f5bf4f]" />
        <span className="bg-[#63c554]" />
      </div>
      <span className="article-code-language">{displayLanguage}</span>
      {actions}
      <button
        type="button"
        className="article-code-copy"
        onClick={copyCode}
        aria-label={copied ? t('markdownCode.copied') : t('markdownCode.copy')}
      >
        {copied ? t('markdownCode.copied') : t('markdownCode.copy')}
      </button>
    </div>
  );
};

export default MarkdownCodeToolbar;
