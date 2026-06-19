import React, { memo, useCallback, useMemo, useState } from 'react';
import ReactMarkdown from 'react-markdown';
import remarkGfm from 'remark-gfm';
import remarkMath from 'remark-math';
import rehypeKatex from 'rehype-katex';
import 'katex/dist/katex.min.css';
import ImageLightbox, { LightboxImage } from './ImageLightbox';
import { createMarkdownComponents } from './MarkdownCode';
import { extractMarkdownHeadings, type MarkdownHeading } from '../utils/markdownHeadings';

type MarkdownRendererProps = {
  content: string;
  className?: string;
  headingIdPrefix?: string;
  /** 外部已提取的 headings；传入可避免渲染器内部重复提取。 */
  headings?: MarkdownHeading[];
};

const MarkdownRenderer: React.FC<MarkdownRendererProps> = ({ content, className = '', headingIdPrefix = '', headings }) => {
  const [lightboxImage, setLightboxImage] = useState<LightboxImage | null>(null);
  const openLightbox = useCallback((image: LightboxImage) => setLightboxImage(image), []);
  const closeLightbox = useCallback(() => setLightboxImage(null), []);
  // 外部传入 headings 时复用，否则内部按 content 提取一次。
  const resolvedHeadings = useMemo(
    () => headings ?? extractMarkdownHeadings(content, 3),
    [headings, content],
  );
  const headingIdByLine = useMemo(() => {
    return resolvedHeadings.reduce<Record<string, string>>((map, heading) => {
      map[`${heading.depth}:${heading.line}`] = `${headingIdPrefix}${heading.id}`;
      return map;
    }, {});
  }, [resolvedHeadings, headingIdPrefix]);
  const components = useMemo(
    () => createMarkdownComponents({ onImageClick: openLightbox, headingIdByLine }),
    [headingIdByLine, openLightbox],
  );

  return (
    <>
      <div className={`article-markdown prose prose-stone max-w-none font-serif ${className}`.trim()}>
        <ReactMarkdown
          remarkPlugins={[remarkGfm, remarkMath]}
          rehypePlugins={[
            rehypeKatex,
          ]}
          components={components}
        >
          {content}
        </ReactMarkdown>
      </div>
      <ImageLightbox image={lightboxImage} onClose={closeLightbox} />
    </>
  );
};

export default memo(MarkdownRenderer);
