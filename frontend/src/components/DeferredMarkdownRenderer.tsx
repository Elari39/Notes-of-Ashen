import { lazy, Suspense } from 'react';
import PagePendingState from './RoutePending';
import type { MarkdownRendererProps } from './MarkdownRenderer';
import { containsMath } from '../utils/markdownMath';

const MarkdownRenderer = lazy(() => import('./MarkdownRenderer'));
// KaTeX（含 CSS）体积大，仅在内容可能包含数学公式时加载带公式支持的渲染器。
const MarkdownRendererWithMath = lazy(() => import('./MarkdownRendererWithMath'));

const DeferredMarkdownRenderer = (props: MarkdownRendererProps) => {
  if (!props.content.trim()) {
    return null;
  }

  const Renderer = containsMath(props.content) ? MarkdownRendererWithMath : MarkdownRenderer;

  return (
    <Suspense fallback={<PagePendingState variant="inline" className="my-4" />}>
      <Renderer {...props} />
    </Suspense>
  );
};

export default DeferredMarkdownRenderer;
