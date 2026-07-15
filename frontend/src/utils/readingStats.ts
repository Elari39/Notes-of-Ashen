export interface ReadingStats { wordCount: number; readingTimeMinutes: number; }

export const getReadingStats = (markdown: string): ReadingStats => {
  let text = markdown
    .replace(/^---\s*\n[\s\S]*?\n---\s*\n/, '')
    .replace(/!\[([^\]]*)\]\([^)]*\)/g, '$1')
    .replace(/\[([^\]]+)\]\([^)]*\)/g, '$1')
    .replace(/<[^>]+>/g, ' ')
    .replace(/https?:\/\/\S+/g, ' ')
    .replace(/[#*_~`>|[\](){}]/g, ' ');
  const han = text.match(/\p{Script=Han}/gu)?.length ?? 0;
  text = text.replace(/\p{Script=Han}/gu, ' ');
  const words = text.match(/[\p{L}\p{N}]+/gu)?.length ?? 0;
  const wordCount = han + words;
  const readingTimeMinutes = text.trim() === '' ? 0 : Math.max(1, Math.ceil(han / 400 + words / 200));
  return { wordCount, readingTimeMinutes };
};
