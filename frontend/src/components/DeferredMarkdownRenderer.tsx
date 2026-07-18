import { lazy, Suspense } from 'react';
import PagePendingState from './RoutePending';
import type { MarkdownRendererProps } from './MarkdownRenderer';

const MarkdownRenderer = lazy(() => import('./MarkdownRenderer'));

const DeferredMarkdownRenderer = (props: MarkdownRendererProps) => {
  if (!props.content.trim()) {
    return null;
  }

  return (
    <Suspense fallback={<PagePendingState variant="inline" className="my-4" />}>
      <MarkdownRenderer {...props} />
    </Suspense>
  );
};

export default DeferredMarkdownRenderer;
