import React, { memo, useCallback, useMemo, useState } from 'react';
import ReactMarkdown, { type Options as ReactMarkdownOptions } from 'react-markdown';
import remarkGfm from 'remark-gfm';
import ImageLightbox, { LightboxImage } from './ImageLightbox';
import { createMarkdownComponents } from './MarkdownCode';
import { extractMarkdownHeadings, type MarkdownHeading } from '../utils/markdownHeadings';

export type MarkdownRendererProps = {
  content: string;
  className?: string;
  headingIdPrefix?: string;
  /** 外部已提取的 headings；传入可避免渲染器内部重复提取。 */
  headings?: MarkdownHeading[];
  /** 追加的 remark 插件（如数学公式）；KaTeX 由 MarkdownRendererWithMath 按需注入。 */
  extraRemarkPlugins?: ReactMarkdownOptions['remarkPlugins'];
  /** 追加的 rehype 插件；与 extraRemarkPlugins 配套使用。 */
  extraRehypePlugins?: ReactMarkdownOptions['rehypePlugins'];
};

const MarkdownRenderer: React.FC<MarkdownRendererProps> = ({
  content,
  className = '',
  headingIdPrefix = '',
  headings,
  extraRemarkPlugins,
  extraRehypePlugins,
}) => {
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
  const remarkPlugins = useMemo<NonNullable<ReactMarkdownOptions['remarkPlugins']>>(
    () => [remarkGfm, ...(extraRemarkPlugins ?? [])],
    [extraRemarkPlugins],
  );
  const rehypePlugins = useMemo<NonNullable<ReactMarkdownOptions['rehypePlugins']>>(
    () => [...(extraRehypePlugins ?? [])],
    [extraRehypePlugins],
  );

  return (
    <>
      <div className={`article-markdown prose prose-stone max-w-none font-sans ${className}`.trim()}>
        <ReactMarkdown
          remarkPlugins={remarkPlugins}
          rehypePlugins={rehypePlugins}
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
