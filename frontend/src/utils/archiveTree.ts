import type { Article } from '../types';

export type ArchiveArticle = Pick<Article, 'id' | 'title' | 'publishedAt' | 'createdAt'>;

export interface ArchiveDayNode {
  key: string;
  day: number;
  articles: ArchiveArticle[];
}

export interface ArchiveMonthNode {
  key: string;
  month: number;
  articleCount: number;
  days: ArchiveDayNode[];
}

export interface ArchiveYearNode {
  key: string;
  year: number;
  articleCount: number;
  months: ArchiveMonthNode[];
}

export interface ArchiveTree {
  years: ArchiveYearNode[];
  undated: ArchiveArticle[];
  articleCount: number;
  defaultExpandedKeys: string[];
}

interface DatedArchiveArticle {
  article: ArchiveArticle;
  date: Date;
  timestamp: number;
}

export const buildArchiveTree = (articles: readonly ArchiveArticle[]): ArchiveTree => {
  const uniqueArticles = [...new Map(articles.map((article) => [article.id, article])).values()];
  const dated: DatedArchiveArticle[] = [];
  const undated: ArchiveArticle[] = [];

  uniqueArticles.forEach((article) => {
    const date = resolveArticleDate(article);
    if (!date) {
      undated.push(article);
      return;
    }
    dated.push({ article, date, timestamp: date.getTime() });
  });

  dated.sort((left, right) => right.timestamp - left.timestamp || right.article.id - left.article.id);
  undated.sort((left, right) => right.id - left.id);

  const yearMap = new Map<number, Map<number, Map<number, ArchiveArticle[]>>>();
  dated.forEach(({ article, date }) => {
    const year = date.getFullYear();
    const month = date.getMonth() + 1;
    const day = date.getDate();
    const monthMap = yearMap.get(year) ?? new Map<number, Map<number, ArchiveArticle[]>>();
    const dayMap = monthMap.get(month) ?? new Map<number, ArchiveArticle[]>();
    const dayArticles = dayMap.get(day) ?? [];
    dayArticles.push(article);
    dayMap.set(day, dayArticles);
    monthMap.set(month, dayMap);
    yearMap.set(year, monthMap);
  });

  const years = [...yearMap.entries()]
    .sort(([left], [right]) => right - left)
    .map(([year, monthMap]): ArchiveYearNode => {
      const months = [...monthMap.entries()]
        .sort(([left], [right]) => right - left)
        .map(([month, dayMap]): ArchiveMonthNode => {
          const days = [...dayMap.entries()]
            .sort(([left], [right]) => right - left)
            .map(([day, dayArticles]): ArchiveDayNode => ({
              key: `day:${year}-${padDatePart(month)}-${padDatePart(day)}`,
              day,
              articles: dayArticles,
            }));
          return {
            key: `month:${year}-${padDatePart(month)}`,
            month,
            articleCount: days.reduce((total, node) => total + node.articles.length, 0),
            days,
          };
        });
      return {
        key: `year:${year}`,
        year,
        articleCount: months.reduce((total, node) => total + node.articleCount, 0),
        months,
      };
    });

  const latestYear = years[0];
  const latestMonth = latestYear?.months[0];
  const latestDay = latestMonth?.days[0];
  const defaultExpandedKeys = latestYear && latestMonth && latestDay
    ? [latestYear.key, latestMonth.key, latestDay.key]
    : (undated.length > 0 ? ['undated'] : []);

  return {
    years,
    undated,
    articleCount: uniqueArticles.length,
    defaultExpandedKeys,
  };
};

const resolveArticleDate = (article: ArchiveArticle): Date | null => {
  for (const value of [article.publishedAt, article.createdAt]) {
    if (!value) {
      continue;
    }
    const date = new Date(value);
    if (Number.isFinite(date.getTime())) {
      return date;
    }
  }
  return null;
};

const padDatePart = (value: number) => String(value).padStart(2, '0');
