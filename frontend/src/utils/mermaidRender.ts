const mermaidErrorTextPattern = /<text\b[^>]*\bclass=["']error-text["']/i;

export const hasMermaidRenderError = (svg: string): boolean => (
  mermaidErrorTextPattern.test(svg) || /Syntax error in text/i.test(svg)
);
