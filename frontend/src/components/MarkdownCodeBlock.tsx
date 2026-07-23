import { useEffect, useRef, useState } from 'react';
import type { CSSProperties } from 'react';
import SyntaxHighlighter from 'react-syntax-highlighter/dist/esm/prism-light';
import { translate } from '../i18n';
import { usePreferenceStore } from '../store/preferences';

type PrismLanguage = Parameters<typeof SyntaxHighlighter.registerLanguage>[1];
type PrismLanguageModule = { default: PrismLanguage };

type SupportedLanguage =
  | 'bash'
  | 'c'
  | 'cpp'
  | 'csharp'
  | 'css'
  | 'docker'
  | 'go'
  | 'ini'
  | 'java'
  | 'javascript'
  | 'jsx'
  | 'json'
  | 'kotlin'
  | 'markdown'
  | 'markup'
  | 'powershell'
  | 'python'
  | 'rust'
  | 'sql'
  | 'toml'
  | 'tsx'
  | 'typescript'
  | 'yaml';

const languageLoaders: Record<SupportedLanguage, () => Promise<PrismLanguageModule>> = {
  bash: () => import('react-syntax-highlighter/dist/esm/languages/prism/bash'),
  c: () => import('react-syntax-highlighter/dist/esm/languages/prism/c'),
  cpp: () => import('react-syntax-highlighter/dist/esm/languages/prism/cpp'),
  csharp: () => import('react-syntax-highlighter/dist/esm/languages/prism/csharp'),
  css: () => import('react-syntax-highlighter/dist/esm/languages/prism/css'),
  docker: () => import('react-syntax-highlighter/dist/esm/languages/prism/docker'),
  go: () => import('react-syntax-highlighter/dist/esm/languages/prism/go'),
  ini: () => import('react-syntax-highlighter/dist/esm/languages/prism/ini'),
  java: () => import('react-syntax-highlighter/dist/esm/languages/prism/java'),
  javascript: () => import('react-syntax-highlighter/dist/esm/languages/prism/javascript'),
  jsx: () => import('react-syntax-highlighter/dist/esm/languages/prism/jsx'),
  json: () => import('react-syntax-highlighter/dist/esm/languages/prism/json'),
  kotlin: () => import('react-syntax-highlighter/dist/esm/languages/prism/kotlin'),
  markdown: () => import('react-syntax-highlighter/dist/esm/languages/prism/markdown'),
  markup: () => import('react-syntax-highlighter/dist/esm/languages/prism/markup'),
  powershell: () => import('react-syntax-highlighter/dist/esm/languages/prism/powershell'),
  python: () => import('react-syntax-highlighter/dist/esm/languages/prism/python'),
  rust: () => import('react-syntax-highlighter/dist/esm/languages/prism/rust'),
  sql: () => import('react-syntax-highlighter/dist/esm/languages/prism/sql'),
  toml: () => import('react-syntax-highlighter/dist/esm/languages/prism/toml'),
  tsx: () => import('react-syntax-highlighter/dist/esm/languages/prism/tsx'),
  typescript: () => import('react-syntax-highlighter/dist/esm/languages/prism/typescript'),
  yaml: () => import('react-syntax-highlighter/dist/esm/languages/prism/yaml'),
};

const languageAliases: Record<string, SupportedLanguage> = {
  bash: 'bash',
  sh: 'bash',
  shell: 'bash',
  c: 'c',
  cpp: 'cpp',
  csharp: 'csharp',
  css: 'css',
  docker: 'docker',
  go: 'go',
  golang: 'go',
  ini: 'ini',
  java: 'java',
  javascript: 'javascript',
  js: 'javascript',
  jsx: 'jsx',
  json: 'json',
  kotlin: 'kotlin',
  markdown: 'markdown',
  md: 'markdown',
  markup: 'markup',
  powershell: 'powershell',
  python: 'python',
  py: 'python',
  rust: 'rust',
  sql: 'sql',
  toml: 'toml',
  tsx: 'tsx',
  typescript: 'typescript',
  ts: 'typescript',
  yaml: 'yaml',
  yml: 'yaml',
};

const loadedLanguages = new Set<SupportedLanguage>();
const languageLoads = new Map<SupportedLanguage, Promise<void>>();

const loadLanguage = (language: SupportedLanguage): Promise<void> => {
  if (loadedLanguages.has(language)) {
    return Promise.resolve();
  }

  const inFlight = languageLoads.get(language);
  if (inFlight) {
    return inFlight;
  }

  const load = languageLoaders[language]().then(({ default: grammar }) => {
    for (const [alias, supportedLanguage] of Object.entries(languageAliases)) {
      if (supportedLanguage === language) {
        SyntaxHighlighter.registerLanguage(alias, grammar);
      }
    }
    loadedLanguages.add(language);
  }).finally(() => {
    languageLoads.delete(language);
  });
  languageLoads.set(language, load);
  return load;
};

const codeFontFamily = '"JetBrains Mono", "SFMono-Regular", Consolas, "Liberation Mono", Menlo, monospace';
const codeBlockStyle: CSSProperties = {
  margin: 0,
  padding: '1.25rem 1.4rem',
  background: 'var(--code-bg)',
  color: 'var(--code-ink)',
  fontSize: '0.92rem',
  lineHeight: 1.72,
  overflowX: 'auto',
};
const codeTagStyle: CSSProperties = {
  color: 'var(--code-ink)',
  fontFamily: codeFontFamily,
  fontWeight: 400,
};

type MarkdownCodeBlockProps = {
  code: string;
  language: string;
  syntaxTheme: Record<string, CSSProperties>;
};

const MarkdownCodeBlock: React.FC<MarkdownCodeBlockProps> = ({ code, language, syntaxTheme }) => {
  const [copied, setCopied] = useState(false);
  const timerRef = useRef<number | undefined>(undefined);
  const uiLanguage = usePreferenceStore((state) => state.language);
  const t = (key: Parameters<typeof translate>[1]) => translate(uiLanguage, key);
  const displayLanguage = language === 'text' ? 'plain text' : language;
  const syntaxLanguage = languageAliases[language.trim().toLowerCase()];
  const [isSyntaxReady, setIsSyntaxReady] = useState(() => Boolean(syntaxLanguage && loadedLanguages.has(syntaxLanguage)));

  useEffect(() => {
    let active = true;
    if (!syntaxLanguage) {
      setIsSyntaxReady(false);
      return () => {
        active = false;
      };
    }

    setIsSyntaxReady(loadedLanguages.has(syntaxLanguage));
    void loadLanguage(syntaxLanguage)
      .then(() => {
        if (active) {
          setIsSyntaxReady(true);
        }
      })
      .catch(() => {
        if (active) {
          // 未知或加载失败时保持普通代码块，文章正文仍可完整阅读。
          setIsSyntaxReady(false);
        }
      });

    return () => {
      active = false;
    };
  }, [syntaxLanguage]);

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
          aria-label={copied ? t('markdownCode.copied') : t('markdownCode.copy')}
        >
          {copied ? t('markdownCode.copied') : t('markdownCode.copy')}
        </button>
      </div>
      {syntaxLanguage && isSyntaxReady ? (
        <SyntaxHighlighter
          style={syntaxTheme}
          language={syntaxLanguage}
          PreTag="div"
          className="article-code-block"
          codeTagProps={{ style: codeTagStyle }}
          customStyle={codeBlockStyle}
        >
          {code}
        </SyntaxHighlighter>
      ) : (
        <pre className="article-code-block" style={codeBlockStyle}>
          <code style={codeTagStyle}>{code}</code>
        </pre>
      )}
    </div>
  );
};

export default MarkdownCodeBlock;
