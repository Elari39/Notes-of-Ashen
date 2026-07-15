import React, { useEffect, useMemo, useState } from 'react';
import { Link } from 'react-router-dom';
import { getArticles } from '../api/article';
import { getCategories } from '../api/category';
import { getTags } from '../api/tag';
import InlineNotice from '../components/InlineNotice';
import { PreloadLink } from '../components/PreloadLink';
import Skeleton from '../components/Skeleton';
import Button from '../components/ui/Button';
import EmptyState from '../components/ui/EmptyState';
import { formatText, getDateLocale, translate } from '../i18n';
import { routeLoaders } from '../routes/lazyRoutes';
import { usePreferenceStore, type Language } from '../store/preferences';
import type { Article, Category, Tag } from '../types';
import {
  buildArchiveTree,
  type ArchiveArticle,
  type ArchiveDayNode,
  type ArchiveMonthNode,
  type ArchiveYearNode,
} from '../utils/archiveTree';

const ARCHIVE_PAGE_SIZE = 100;

const Archive: React.FC = () => {
  const language = usePreferenceStore((state) => state.language);
  const [articles, setArticles] = useState<Article[]>([]);
  const [categories, setCategories] = useState<Category[]>([]);
  const [tags, setTags] = useState<Tag[]>([]);
  const [historyLoading, setHistoryLoading] = useState(true);
  const [filtersLoading, setFiltersLoading] = useState(true);
  const [historyError, setHistoryError] = useState(false);
  const [filtersError, setFiltersError] = useState(false);
  const [retryVersion, setRetryVersion] = useState(0);
  const [expandedKeys, setExpandedKeys] = useState<Set<string>>(() => new Set());
  const t = (key: Parameters<typeof translate>[1]) => translate(language, key);
  const archiveTree = useMemo(() => buildArchiveTree(articles), [articles]);

  useEffect(() => {
    let active = true;
    const controller = new AbortController();

    const loadHistory = async () => {
      setHistoryLoading(true);
      setHistoryError(false);
      try {
        const items = await getAllPublishedArticles(controller.signal);
        if (active) {
          setArticles(items);
        }
      } catch {
        if (active && !controller.signal.aborted) {
          setHistoryError(true);
        }
      } finally {
        if (active) {
          setHistoryLoading(false);
        }
      }
    };

    const loadFilters = async () => {
      setFiltersLoading(true);
      setFiltersError(false);
      const [categoryResult, tagResult] = await Promise.allSettled([
        getCategories({ size: ARCHIVE_PAGE_SIZE }),
        getTags({ size: ARCHIVE_PAGE_SIZE }),
      ]);
      if (!active) {
        return;
      }
      if (categoryResult.status === 'fulfilled') {
        setCategories(categoryResult.value.data.items || []);
      }
      if (tagResult.status === 'fulfilled') {
        setTags(tagResult.value.data.items || []);
      }
      setFiltersError(categoryResult.status === 'rejected' || tagResult.status === 'rejected');
      setFiltersLoading(false);
    };

    void loadHistory();
    void loadFilters();
    return () => {
      active = false;
      controller.abort();
    };
  }, [retryVersion]);

  useEffect(() => {
    setExpandedKeys(new Set(archiveTree.defaultExpandedKeys));
  }, [archiveTree]);

  const toggleExpanded = (key: string) => {
    setExpandedKeys((current) => {
      const next = new Set(current);
      if (next.has(key)) {
        next.delete(key);
      } else {
        next.add(key);
      }
      return next;
    });
  };

  const handleRetry = () => setRetryVersion((version) => version + 1);
  const hasFilterData = categories.length > 0 || tags.length > 0;

  return (
    <div className="editorial-container w-full space-y-12">
      <header className="max-w-3xl py-6 md:py-10">
        <p className="editorial-kicker">{t('archive.kicker')}</p>
        <h1 className="mt-5 editorial-page-title">{t('nav.archive')}</h1>
      </header>

      <section aria-labelledby="archive-history-title" className="overflow-hidden rounded-xl border border-hairline bg-paper shadow-sm">
        <div className="border-b border-hairline px-5 py-7 sm:px-7 md:px-8 md:py-8">
          <div aria-hidden="true" className="mb-5 flex items-center gap-3">
            <span className="h-px w-10 bg-ochre" />
            <span className="h-1.5 w-1.5 rounded-full bg-ochre" />
            <span className="h-px flex-1 bg-hairline" />
          </div>
          <div className="flex flex-col gap-3 sm:flex-row sm:items-end sm:justify-between">
            <div>
              <p className="text-xs font-medium uppercase tracking-[0.18em] text-ochre">{t('archive.historyKicker')}</p>
              <h2 id="archive-history-title" className="mt-3 font-display text-3xl font-normal tracking-[-0.02em] text-ink md:text-4xl">
                {t('archive.historyTitle')}
              </h2>
              <p className="mt-3 max-w-2xl text-sm leading-7 text-body">{t('archive.historyDescription')}</p>
            </div>
            {archiveTree.articleCount > 0 && (
              <span className="shrink-0 self-start rounded-full border border-hairline bg-surface-soft px-3 py-1.5 font-mono text-xs text-muted sm:self-auto">
                {formatArchiveArticleCount(language, archiveTree.articleCount)}
              </span>
            )}
          </div>
        </div>

        <div className="p-4 sm:p-6 md:p-8">
          <InlineNotice
            message={historyError ? t('archive.historyLoadError') : ''}
            className="mb-5"
            action={(
              <Button type="button" variant="ghost" size="sm" onClick={handleRetry}>
                {t('common.retry')}
              </Button>
            )}
          />

          {historyLoading && articles.length === 0 && <ArchiveTreeSkeleton />}
          {historyLoading && articles.length > 0 && (
            <p role="status" className="mb-4 text-xs tracking-wide text-muted">{t('archive.historyLoading')}</p>
          )}
          {!historyLoading && !historyError && archiveTree.articleCount === 0 && (
            <div className="rounded-xl border border-dashed border-hairline bg-surface-soft px-6 py-14 text-center">
              <span aria-hidden="true" className="mx-auto mb-5 block h-2 w-2 rounded-full bg-ochre" />
              <p className="font-display text-2xl text-ink">{t('archive.emptyHistory')}</p>
            </div>
          )}
          {archiveTree.articleCount > 0 && (
            <ArchiveTreeView
              years={archiveTree.years}
              undated={archiveTree.undated}
              expandedKeys={expandedKeys}
              onToggle={toggleExpanded}
              language={language}
            />
          )}
        </div>
      </section>

      <section aria-label={`${t('archive.titleCategories')} / ${t('archive.titleTags')}`} className="editorial-card">
        <InlineNotice
          message={filtersError ? t('archive.filtersLoadError') : ''}
          className="mb-6"
          action={(
            <Button type="button" variant="ghost" size="sm" onClick={handleRetry}>
              {t('common.retry')}
            </Button>
          )}
        />
        {filtersLoading && !hasFilterData ? (
          <div className="grid gap-8 md:grid-cols-2">
            {[0, 1].map((section) => (
              <div key={section}>
                <Skeleton className="mb-7 h-8 w-28" />
                <div className="flex flex-wrap gap-3">
                  {Array.from({ length: 6 }).map((_, index) => (
                    <Skeleton key={index} className="h-11 w-24" />
                  ))}
                </div>
              </div>
            ))}
          </div>
        ) : (
          <div className="grid gap-10 md:grid-cols-2 md:gap-0">
            <div className="md:border-r md:border-hairline md:pr-8">
              <h2 className="mb-7 font-display text-3xl text-ink md:text-4xl">{t('archive.titleCategories')}</h2>
              {categories.length === 0 ? (
                <EmptyState illustration="leaf" title={t('common.noCategory')} className="bg-paper" />
              ) : (
                <div className="flex flex-wrap gap-3">
                  {categories.map((category) => (
                    <Link
                      key={category.id}
                      to={`/?categoryId=${category.id}`}
                      className="inline-flex min-h-11 items-center rounded-md border border-hairline bg-paper px-4 py-2 text-sm font-medium text-ink transition-colors duration-base hover:border-ink focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-ochre"
                    >
                      {category.name}
                    </Link>
                  ))}
                </div>
              )}
            </div>

            <div className="md:pl-8">
              <h2 className="mb-7 font-display text-3xl text-ink md:text-4xl">{t('archive.titleTags')}</h2>
              {tags.length === 0 ? (
                <EmptyState illustration="leaf" title={t('common.noTag')} className="bg-paper" />
              ) : (
                <div className="flex flex-wrap gap-3">
                  {tags.map((tag) => (
                    <Link
                      key={tag.id}
                      to={`/?tagId=${tag.id}`}
                      className="inline-flex min-h-11 items-center rounded-full border border-hairline bg-paper px-4 py-2 text-sm text-ink transition-colors duration-base before:mr-1 before:opacity-40 before:content-['#'] hover:border-ochre hover:text-ochre focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-ochre"
                    >
                      {tag.name}
                    </Link>
                  ))}
                </div>
              )}
            </div>
          </div>
        )}
        {filtersLoading && hasFilterData && (
          <p role="status" className="mt-5 text-xs text-muted">{t('common.loadingArchive')}</p>
        )}
      </section>
    </div>
  );
};

interface ArchiveTreeViewProps {
  years: ArchiveYearNode[];
  undated: ArchiveArticle[];
  expandedKeys: Set<string>;
  onToggle: (key: string) => void;
  language: Language;
}

const ArchiveTreeView: React.FC<ArchiveTreeViewProps> = ({ years, undated, expandedKeys, onToggle, language }) => {
  const t = (key: Parameters<typeof translate>[1]) => translate(language, key);
  return (
    <ul aria-label={t('archive.historyTitle')} className="space-y-5 text-sm">
      {years.map((year) => (
        <YearBranch
          key={year.key}
          node={year}
          expandedKeys={expandedKeys}
          onToggle={onToggle}
          language={language}
        />
      ))}
      {undated.length > 0 && (
        <DisclosureBranch
          nodeKey="undated"
          label={t('archive.undated')}
          count={undated.length}
          level="undated"
          expanded={expandedKeys.has('undated')}
          onToggle={onToggle}
          language={language}
        >
          <ArticleLeaves articles={undated} />
        </DisclosureBranch>
      )}
    </ul>
  );
};

interface BranchProps {
  expandedKeys: Set<string>;
  onToggle: (key: string) => void;
  language: Language;
}

const YearBranch: React.FC<BranchProps & { node: ArchiveYearNode }> = ({ node, expandedKeys, onToggle, language }) => (
  <DisclosureBranch
    nodeKey={node.key}
    label={formatArchiveYear(node.year, language)}
    count={node.articleCount}
    level="year"
    expanded={expandedKeys.has(node.key)}
    onToggle={onToggle}
    language={language}
  >
    {node.months.map((month) => (
      <MonthBranch key={month.key} node={month} year={node.year} expandedKeys={expandedKeys} onToggle={onToggle} language={language} />
    ))}
  </DisclosureBranch>
);

const MonthBranch: React.FC<BranchProps & { node: ArchiveMonthNode; year: number }> = ({ node, year, expandedKeys, onToggle, language }) => (
  <DisclosureBranch
    nodeKey={node.key}
    label={formatArchiveMonth(node.month, language)}
    count={node.articleCount}
    level="month"
    expanded={expandedKeys.has(node.key)}
    onToggle={onToggle}
    language={language}
  >
    {node.days.map((day) => (
      <DayBranch key={day.key} node={day} year={year} month={node.month} expandedKeys={expandedKeys} onToggle={onToggle} language={language} />
    ))}
  </DisclosureBranch>
);

const DayBranch: React.FC<BranchProps & { node: ArchiveDayNode; year: number; month: number }> = ({ node, year, month, expandedKeys, onToggle, language }) => (
  <DisclosureBranch
    nodeKey={node.key}
    label={formatArchiveDay(year, month, node.day, language)}
    count={node.articles.length}
    level="day"
    expanded={expandedKeys.has(node.key)}
    onToggle={onToggle}
    language={language}
  >
    <ArticleLeaves articles={node.articles} />
  </DisclosureBranch>
);

type ArchiveBranchLevel = 'year' | 'month' | 'day' | 'undated';

interface DisclosureBranchProps {
  nodeKey: string;
  label: string;
  count: number;
  level: ArchiveBranchLevel;
  expanded: boolean;
  onToggle: (key: string) => void;
  language: Language;
  children: React.ReactNode;
}

const DisclosureBranch: React.FC<DisclosureBranchProps> = ({ nodeKey, label, count, level, expanded, onToggle, language, children }) => {
  const t = (key: Parameters<typeof translate>[1]) => translate(language, key);
  const contentID = `archive-${nodeKey.replace(/[^a-zA-Z0-9_-]/g, '-')}`;
  const actionLabel = formatText(t(expanded ? 'archive.collapse' : 'archive.expand'), { label });
  const isChapter = level === 'year' || level === 'undated';
  const labelClassName = isChapter
    ? 'font-display text-3xl leading-none text-ink sm:text-4xl'
    : level === 'month'
      ? 'font-display text-xl leading-tight text-ink sm:text-2xl'
      : 'font-sans text-sm font-medium text-ink sm:text-base';
  const contentClassName = isChapter
    ? 'space-y-1 border-t border-hairline px-2 pb-3 pt-2 sm:px-4 sm:pb-4 sm:pt-3'
    : 'relative ml-4 space-y-1 border-l border-hairline pl-2 sm:ml-5 sm:pl-4';
  return (
    <li className={isChapter ? 'overflow-hidden rounded-xl border border-hairline bg-paper shadow-xs' : 'relative'}>
      <button
        type="button"
        aria-expanded={expanded}
        aria-controls={contentID}
        aria-label={actionLabel}
        onClick={() => onToggle(nodeKey)}
        className={`group flex min-h-11 w-full items-center gap-3 text-left text-ink transition-colors duration-fast hover:bg-surface-soft focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-ochre ${isChapter ? 'px-4 py-4 sm:px-5 sm:py-5' : 'rounded-md px-2 py-2'}`}
      >
        <ChevronIcon expanded={expanded} />
        <ChronicleMarker level={level} expanded={expanded} />
        <span className={`min-w-0 flex-1 break-words ${labelClassName}`}>{label}</span>
        <span className={`shrink-0 rounded-full border border-hairline bg-surface-soft px-2.5 py-1 font-mono text-[11px] text-muted ${level === 'day' ? 'hidden sm:inline-flex' : 'inline-flex'}`}>
          {formatArchiveArticleCount(language, count)}
        </span>
      </button>
      {expanded && (
        <ul id={contentID} className={contentClassName}>
          {children}
        </ul>
      )}
    </li>
  );
};

const ArticleLeaves: React.FC<{ articles: ArchiveArticle[] }> = ({ articles }) => (
  <>
    {articles.map((article) => (
      <li key={article.id}>
        <PreloadLink
          to={`/article/${article.id}`}
          preload={routeLoaders.articleDetail}
          className="group flex min-h-11 items-start gap-3 rounded-md px-2 py-2.5 text-body transition-colors duration-fast hover:bg-surface-soft hover:text-ochre focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-ochre"
        >
          <PageMarkIcon />
          <span className="min-w-0 flex-1 break-words font-sans leading-6">{article.title}</span>
        </PreloadLink>
      </li>
    ))}
  </>
);

const ArchiveTreeSkeleton = () => (
  <div className="space-y-5" aria-hidden="true">
    {[0, 1].map((chapter) => (
      <div key={chapter} className="rounded-xl border border-hairline bg-paper p-4 sm:p-5">
        <div className="flex min-h-11 items-center gap-3">
          <Skeleton className="h-4 w-4 shrink-0 rounded-full" />
          <Skeleton className="h-3 w-3 shrink-0 rounded-full" />
          <Skeleton className="h-9 w-36" />
          <Skeleton className="ml-auto h-6 w-14 rounded-full" />
        </div>
        {chapter === 0 && (
          <div className="mt-4 space-y-3 border-t border-hairline pt-4">
            <Skeleton className="h-11 w-2/3" />
            <div className="ml-4 space-y-3 border-l border-hairline pl-3">
              <Skeleton className="h-11 w-3/4" />
              <Skeleton className="h-11 w-full" />
            </div>
          </div>
        )}
      </div>
    ))}
  </div>
);

const ChevronIcon = ({ expanded }: { expanded: boolean }) => (
  <svg aria-hidden="true" viewBox="0 0 16 16" className={`h-4 w-4 shrink-0 text-muted transition-transform duration-fast ${expanded ? 'rotate-90' : ''}`} fill="none">
    <path d="M6 3.5L10.5 8L6 12.5" stroke="currentColor" strokeWidth="1.5" strokeLinecap="round" strokeLinejoin="round" />
  </svg>
);

const ChronicleMarker = ({ level, expanded }: { level: ArchiveBranchLevel; expanded: boolean }) => {
  const sizeClassName = level === 'year' || level === 'undated'
    ? 'h-3.5 w-3.5 border-2'
    : level === 'month'
      ? 'h-3 w-3 border-2'
      : 'h-2.5 w-2.5 border';
  return (
    <span
      aria-hidden="true"
      className={`shrink-0 rounded-full border-ochre transition-colors duration-fast ${sizeClassName} ${expanded ? 'bg-ochre' : 'bg-paper group-hover:bg-ochre'}`}
    />
  );
};

const PageMarkIcon = () => (
  <svg aria-hidden="true" viewBox="0 0 16 18" className="mt-1 h-[18px] w-4 shrink-0 text-muted transition-colors duration-fast group-hover:text-ochre" fill="none">
    <path d="M3 2.5H10.5L13 5V15.5H3V2.5Z" stroke="currentColor" strokeWidth="1.2" strokeLinejoin="round" />
    <path d="M10.5 2.5V5H13M5.5 8.5H10.5M5.5 11.5H9" stroke="currentColor" strokeWidth="1.2" strokeLinecap="round" />
  </svg>
);

const getAllPublishedArticles = async (signal: AbortSignal): Promise<Article[]> => {
  const items: Article[] = [];
  let page = 1;
  let hasMore = true;
  while (hasMore) {
    const response = await getArticles({ page, size: ARCHIVE_PAGE_SIZE, status: 'published' }, signal);
    const pageItems = response.data.items || [];
    items.push(...pageItems);
    hasMore = pageItems.length === ARCHIVE_PAGE_SIZE && page * ARCHIVE_PAGE_SIZE < response.data.total;
    if (hasMore) {
      page += 1;
    }
  }
  return items;
};

const formatArchiveYear = (year: number, language: Language) => language === 'zh' ? `${year} 年` : String(year);

const formatArchiveMonth = (month: number, language: Language) => new Intl.DateTimeFormat(getDateLocale(language), {
  month: 'long',
}).format(new Date(2000, month - 1, 1));

const formatArchiveDay = (year: number, month: number, day: number, language: Language) => new Intl.DateTimeFormat(getDateLocale(language), {
  month: 'short',
  day: 'numeric',
}).format(new Date(year, month - 1, day));

const formatArchiveArticleCount = (language: Language, count: number) => formatText(
  translate(language, language === 'en' && count === 1 ? 'archive.articleCountOne' : 'archive.articleCount'),
  { count },
);

export default Archive;
