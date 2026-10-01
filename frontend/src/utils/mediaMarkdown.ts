type MarkdownMedia = { url: string; altText: string; originalName: string };

// The media picker inserts a block image. Keeping paragraph boundaries avoids
// joining an image to a closing code fence, heading or neighbouring text.
export const insertMediaMarkdown = (content: string, start: number, end: number, asset: MarkdownMedia) => {
  const before = content.slice(0, start);
  const after = content.slice(end);
  const alt = (asset.altText || asset.originalName.replace(/\.[^.]+$/, ''))
    .replace(/[\r\n]+/g, ' ').replace(/[\\[\]]/g, '\\$&');
  const leading = before && !before.endsWith('\n\n') ? (before.endsWith('\n') ? '\n' : '\n\n') : '';
  const trailing = after && !after.startsWith('\n\n') ? (after.startsWith('\n') ? '\n' : '\n\n') : '';
  const insertion = `${leading}![${alt}](${asset.url})${trailing}`;
  return { content: `${before}${insertion}${after}`, cursor: before.length + insertion.length };
};
