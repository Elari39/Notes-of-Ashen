/**
 * 判断 Markdown 内容是否可能包含数学公式，决定是否加载 KaTeX chunk。
 *
 * 宽松启发式：匹配行内/块级 `$...$`、`$$...$$` 以及 `\(...\)`、`\[...\]`。
 * 误判（如普通文本中的价格 "$5"）只会多加载一次 katex 资源，不影响渲染正确性；
 * 漏判会导致公式以原文显示，因此判定倾向宽松。
 */
export const containsMath = (content: string): boolean => {
  if (!content) {
    return false;
  }
  return content.includes('$') || /\\[([]/.test(content);
};
