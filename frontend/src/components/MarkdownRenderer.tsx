import React, { useCallback, useMemo, useState } from 'react';
import ReactMarkdown from 'react-markdown';
import remarkGfm from 'remark-gfm';
import remarkMath from 'remark-math';
import rehypeKatex from 'rehype-katex';
import 'katex/dist/katex.min.css';
import ImageLightbox, { LightboxImage } from './ImageLightbox';
import { createMarkdownComponents } from './MarkdownCode';

type MarkdownRendererProps = {
  content: string;
  className?: string;
};

const MarkdownRenderer: React.FC<MarkdownRendererProps> = ({ content, className = '' }) => {
  const [lightboxImage, setLightboxImage] = useState<LightboxImage | null>(null);
  const openLightbox = useCallback((image: LightboxImage) => setLightboxImage(image), []);
  const closeLightbox = useCallback(() => setLightboxImage(null), []);
  const components = useMemo(
    () => createMarkdownComponents({ onImageClick: openLightbox }),
    [openLightbox],
  );

  return (
    <>
      <div className={`article-markdown prose prose-stone max-w-none font-serif ${className}`.trim()}>
        <ReactMarkdown
          remarkPlugins={[remarkGfm, remarkMath]}
          rehypePlugins={[rehypeKatex]}
          components={components}
        >
          {content}
        </ReactMarkdown>
      </div>
      <ImageLightbox image={lightboxImage} onClose={closeLightbox} />
    </>
  );
};

export default MarkdownRenderer;
