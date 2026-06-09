export type MarkdownHeading = {
  id: string;
  depth: number;
  line: number;
  title: string;
};

export const extractMarkdownHeadings = (content: string, maxDepth = 3): MarkdownHeading[] => {
  const counts = new Map<string, number>();
  let inFence = false;

  return content
    .split(/\r?\n/)
    .map((line, index) => {
      if (/^\s*(```|~~~)/.test(line)) {
        inFence = !inFence;
        return { match: null, line: index + 1 };
      }
      return {
        match: inFence ? null : line.match(/^(#{1,6})\s+(.+?)\s*#*\s*$/),
        line: index + 1,
      };
    })
    .filter((item): item is { match: RegExpMatchArray; line: number } => Boolean(item.match))
    .filter((item) => item.match[1].length <= maxDepth)
    .map((match, index) => {
      const title = stripHeadingMarkdown(match.match[2]);
      const base = headingSlug(title) || `heading-${index + 1}`;
      const count = counts.get(base) || 0;
      counts.set(base, count + 1);
      return {
        id: count === 0 ? base : `${base}-${count}`,
        depth: match.match[1].length,
        line: match.line,
        title,
      };
    })
    .filter((heading) => heading.title);
};

export const headingSlug = (value: string) => value
  .trim()
  .toLowerCase()
  .replace(/[^\p{Letter}\p{Number}\p{Mark}\s-]/gu, '')
  .replace(/\s+/g, '-');

const stripHeadingMarkdown = (value: string) => value
  .replace(/\[([^\]]+)\]\([^)]+\)/g, '$1')
  .replace(/[`*_~]/g, '')
  .trim();
