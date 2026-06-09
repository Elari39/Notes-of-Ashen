import type { CSSProperties, KeyboardEvent, MouseEvent } from 'react';
import type { Components } from 'react-markdown';
import SyntaxHighlighter from 'react-syntax-highlighter/dist/esm/prism-light';
import bash from 'react-syntax-highlighter/dist/esm/languages/prism/bash';
import css from 'react-syntax-highlighter/dist/esm/languages/prism/css';
import go from 'react-syntax-highlighter/dist/esm/languages/prism/go';
import javascript from 'react-syntax-highlighter/dist/esm/languages/prism/javascript';
import jsx from 'react-syntax-highlighter/dist/esm/languages/prism/jsx';
import json from 'react-syntax-highlighter/dist/esm/languages/prism/json';
import markdown from 'react-syntax-highlighter/dist/esm/languages/prism/markdown';
import python from 'react-syntax-highlighter/dist/esm/languages/prism/python';
import sql from 'react-syntax-highlighter/dist/esm/languages/prism/sql';
import tsx from 'react-syntax-highlighter/dist/esm/languages/prism/tsx';
import typescript from 'react-syntax-highlighter/dist/esm/languages/prism/typescript';
import yaml from 'react-syntax-highlighter/dist/esm/languages/prism/yaml';
import type { LightboxImage } from './ImageLightbox';
import MarkdownCodeBlock from './MarkdownCodeBlock';

SyntaxHighlighter.registerLanguage('bash', bash);
SyntaxHighlighter.registerLanguage('sh', bash);
SyntaxHighlighter.registerLanguage('shell', bash);
SyntaxHighlighter.registerLanguage('css', css);
SyntaxHighlighter.registerLanguage('go', go);
SyntaxHighlighter.registerLanguage('golang', go);
SyntaxHighlighter.registerLanguage('jsx', jsx);
SyntaxHighlighter.registerLanguage('javascript', javascript);
SyntaxHighlighter.registerLanguage('js', javascript);
SyntaxHighlighter.registerLanguage('json', json);
SyntaxHighlighter.registerLanguage('markdown', markdown);
SyntaxHighlighter.registerLanguage('md', markdown);
SyntaxHighlighter.registerLanguage('python', python);
SyntaxHighlighter.registerLanguage('py', python);
SyntaxHighlighter.registerLanguage('sql', sql);
SyntaxHighlighter.registerLanguage('tsx', tsx);
SyntaxHighlighter.registerLanguage('typescript', typescript);
SyntaxHighlighter.registerLanguage('ts', typescript);
SyntaxHighlighter.registerLanguage('yaml', yaml);
SyntaxHighlighter.registerLanguage('yml', yaml);

const codeFontFamily = '"SFMono-Regular", Consolas, "Liberation Mono", Menlo, monospace';

const syntaxTheme: Record<string, CSSProperties> = {
  'code[class*="language-"]': {
    color: 'var(--code-ink)',
    fontFamily: codeFontFamily,
    textShadow: 'none',
  },
  'pre[class*="language-"]': {
    color: 'var(--code-ink)',
    background: 'var(--code-bg)',
    fontFamily: codeFontFamily,
    textShadow: 'none',
  },
  comment: { color: 'var(--code-comment)' },
  prolog: { color: 'var(--code-comment)' },
  doctype: { color: 'var(--code-comment)' },
  cdata: { color: 'var(--code-comment)' },
  punctuation: { color: 'var(--code-muted)' },
  property: { color: 'var(--code-ochre)' },
  tag: { color: 'var(--code-ochre)' },
  boolean: { color: 'var(--code-ochre)' },
  number: { color: 'var(--code-ochre)' },
  constant: { color: 'var(--code-ochre)' },
  symbol: { color: 'var(--code-ochre)' },
  deleted: { color: 'var(--code-ochre)' },
  selector: { color: 'var(--code-green)' },
  'attr-name': { color: 'var(--code-green)' },
  string: { color: 'var(--code-green)' },
  char: { color: 'var(--code-green)' },
  builtin: { color: 'var(--code-green)' },
  inserted: { color: 'var(--code-green)' },
  operator: { color: 'var(--code-muted)' },
  entity: { color: 'var(--code-amber)' },
  url: { color: 'var(--code-amber)' },
  variable: { color: 'var(--code-amber)' },
  atrule: { color: 'var(--code-amber)' },
  'attr-value': { color: 'var(--code-amber)' },
  function: { color: 'var(--code-blue)' },
  'class-name': { color: 'var(--code-blue)' },
  keyword: { color: 'var(--code-purple)', fontWeight: 600 },
  regex: { color: 'var(--code-amber)' },
  important: { color: 'var(--code-purple)', fontWeight: 700 },
  bold: { fontWeight: 700 },
  italic: { fontStyle: 'italic' },
};

const languageAliases: Record<string, string> = {
  golang: 'go',
  js: 'javascript',
  md: 'markdown',
  py: 'python',
  shell: 'bash',
  sh: 'bash',
  ts: 'typescript',
  yml: 'yaml',
};

const getLanguage = (className?: string) => {
  const match = /language-([\w-]+)/.exec(className || '');
  if (!match) {
    return '';
  }
  const language = match[1].toLowerCase();
  return languageAliases[language] || language;
};

type MarkdownComponentOptions = {
  onImageClick?: (image: LightboxImage) => void;
  headingIdByLine?: Record<string, string>;
};

export const createMarkdownComponents = ({ onImageClick, headingIdByLine }: MarkdownComponentOptions = {}): Components => {
  const headingId = (depth: number, node: unknown) => {
    const line = typeof node === 'object' && node !== null && 'position' in node
      ? (node as { position?: { start?: { line?: number } } }).position?.start?.line
      : undefined;
    return line ? headingIdByLine?.[`${depth}:${line}`] : undefined;
  };

  return {
    pre({ children }) {
      return <>{children}</>;
    },
    h1({ children, node, ...props }) {
      const id = headingId(1, node);
      return <h1 {...props} id={id}>{children}</h1>;
    },
    h2({ children, node, ...props }) {
      const id = headingId(2, node);
      return <h2 {...props} id={id}>{children}</h2>;
    },
    h3({ children, node, ...props }) {
      const id = headingId(3, node);
      return <h3 {...props} id={id}>{children}</h3>;
    },
    table({ children, ...props }) {
      return (
        <div className="article-table-wrap">
          <table {...props}>{children}</table>
        </div>
      );
    },
    code({ className, children, node, ...props }) {
      const language = getLanguage(className);
      const code = String(children).replace(/\n$/, '');
      const isBlock = Boolean(language) || code.includes('\n') || Boolean(node?.position && node.position.start.line !== node.position.end.line);

      return isBlock ? (
        <MarkdownCodeBlock code={code} language={language || 'text'} syntaxTheme={syntaxTheme} />
      ) : (
        <code {...props} className="article-inline-code">
          {children}
        </code>
      );
    },
    img({ src, alt, title, className, node, ...props }) {
      void node;

      const imageSrc = typeof src === 'string' ? src : '';
      const imageAlt = typeof alt === 'string' ? alt : '';
      const imageTitle = typeof title === 'string' ? title : undefined;
      const imageClassName = ['article-image-clickable', className].filter(Boolean).join(' ');

      if (!imageSrc || !onImageClick) {
        return (
          <img
            {...props}
            src={src}
            alt={imageAlt}
            title={imageTitle}
            className={className}
          />
        );
      }

      const openImage = () => {
        onImageClick({
          src: imageSrc,
          alt: imageAlt || imageTitle || '',
        });
      };

      const handleClick = (event: MouseEvent<HTMLImageElement>) => {
        event.preventDefault();
        event.stopPropagation();
        openImage();
      };

      const handleKeyDown = (event: KeyboardEvent<HTMLImageElement>) => {
        if (event.key === 'Enter' || event.key === ' ') {
          event.preventDefault();
          event.stopPropagation();
          openImage();
        }
      };

      return (
        <img
          {...props}
          src={imageSrc}
          alt={imageAlt}
          title={imageTitle}
          className={imageClassName}
          role="button"
          tabIndex={0}
          aria-label={imageAlt ? `查看大图：${imageAlt}` : '查看大图'}
          onClick={handleClick}
          onKeyDown={handleKeyDown}
        />
      );
    },
  };
};

export const markdownComponents = createMarkdownComponents();
