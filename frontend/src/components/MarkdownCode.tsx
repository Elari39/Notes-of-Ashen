import { lazy, Suspense, type CSSProperties, type KeyboardEvent, type MouseEvent } from 'react';
import type { Components, ExtraProps } from 'react-markdown';
import type { LightboxImage } from './ImageLightbox';
import MarkdownTable from './MarkdownTable';
import { usePreferenceStore } from '../store/preferences';
import { resolveMarkdownCodeLanguage } from '../utils/markdownCodeLanguage';

const MarkdownCodeBlock = lazy(() => import('./MarkdownCodeBlock'));

const codeFontFamily = '"JetBrains Mono", "SFMono-Regular", Consolas, "Liberation Mono", Menlo, monospace';

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

const openImageLabel = (language: 'zh' | 'en', alt = '') => {
  if (language === 'en') {
    return alt ? `View full-size image: ${alt}` : 'View full-size image';
  }
  return alt ? `查看大图：${alt}` : '查看大图';
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

  const MarkdownImage = createMarkdownImage(onImageClick);

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
      void props;
      return (
        <MarkdownTable>{children}</MarkdownTable>
      );
    },
    code({ className, children, node, ...props }) {
      const language = resolveMarkdownCodeLanguage(className);
      const code = String(children).replace(/\n$/, '');
      const isBlock = Boolean(language) || code.includes('\n') || Boolean(node?.position && node.position.start.line !== node.position.end.line);

      return isBlock ? (
        <Suspense fallback={<pre className="article-code-block" aria-busy="true"><code>{code}</code></pre>}>
          <MarkdownCodeBlock code={code} language={language || 'text'} syntaxTheme={syntaxTheme} />
        </Suspense>
      ) : (
        <code {...props} className="article-inline-code">
          {children}
        </code>
      );
    },
    img: MarkdownImage,
  };
};

// 独立命名组件以符合 React Hooks 规则；通过闭包捕获 onImageClick。
// 订阅 preference store，使 i18n 切换时图片 aria-label 文案立即重渲。
type MarkdownImageProps = React.ClassAttributes<HTMLImageElement> & React.ImgHTMLAttributes<HTMLImageElement> & ExtraProps;

function createMarkdownImage(onImageClick?: (image: LightboxImage) => void) {
  const MarkdownImage = ({ src, alt, title, className, node, ...rest }: MarkdownImageProps) => {
    void node;
    const language = usePreferenceStore((state) => state.language);

    const imageSrc = typeof src === 'string' ? src : '';
    const imageAlt = typeof alt === 'string' ? alt : '';
    const imageTitle = typeof title === 'string' ? title : undefined;
    const imageClassName = ['article-image-clickable', className].filter(Boolean).join(' ');

    if (!imageSrc || !onImageClick) {
      return (
        <img
          {...rest}
          src={src}
          alt={imageAlt}
          title={imageTitle}
          className={className}
          loading="lazy"
          decoding="async"
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
        {...rest}
        src={imageSrc}
        alt={imageAlt}
        title={imageTitle}
        className={imageClassName}
        loading="lazy"
        decoding="async"
        role="button"
        tabIndex={0}
        aria-label={openImageLabel(language, imageAlt)}
        onClick={handleClick}
        onKeyDown={handleKeyDown}
      />
    );
  };
  return MarkdownImage;
}

export const markdownComponents = createMarkdownComponents();
