import remarkMath from 'remark-math';
import rehypeKatex from 'rehype-katex';
import 'katex/dist/katex.min.css';
import MarkdownRenderer, { type MarkdownRendererProps } from './MarkdownRenderer';

const mathRemarkPlugins = [remarkMath];
const mathRehypePlugins = [rehypeKatex];

/**
 * 带 KaTeX 数学公式支持的 Markdown 渲染器。
 * 通过 DeferredMarkdownRenderer 按内容是否含公式懒加载，
 * 使 katex（含 CSS）只在需要时进入页面。
 */
const MarkdownRendererWithMath = (props: MarkdownRendererProps) => (
  <MarkdownRenderer
    {...props}
    extraRemarkPlugins={mathRemarkPlugins}
    extraRehypePlugins={mathRehypePlugins}
  />
);

export default MarkdownRendererWithMath;
