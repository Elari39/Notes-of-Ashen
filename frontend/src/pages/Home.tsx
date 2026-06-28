import React, { useEffect, useState } from 'react';
import { Link, useSearchParams } from 'react-router-dom';
import Pagination from '../components/Pagination';
import InlineNotice from '../components/InlineNotice';
import PagePendingState from '../components/RoutePending';
import ArticleCardSkeleton from '../components/ArticleCardSkeleton';
import EmptyState from '../components/ui/EmptyState';
import Button from '../components/ui/Button';
import Tag from '../components/ui/Tag';
import { PreloadLink } from '../components/PreloadLink';
import { getErrorMessage } from '../utils/error';
import { getDateLocale, translate } from '../i18n';
import { usePreferenceStore } from '../store/preferences';
import { getArticles } from '../api/article';
import { Article } from '../types';
import { normalizeCoverUrl } from '../utils/cover';
import { useSiteSettingsStore } from '../store/siteSettings';
import { routeLoaders } from '../routes/lazyRoutes';

const Home: React.FC = () => {
  const language = usePreferenceStore((state) => state.language);
  const homeArticleLayout = useSiteSettingsStore((state) => state.homeArticleLayout);
  const [searchParams, setSearchParams] = useSearchParams();
  const [articles, setArticles] = useState<Article[]>([]);
  const [loading, setLoading] = useState(true);
  const [total, setTotal] = useState(0);
  const [error, setError] = useState('');
  const [coverErrors, setCoverErrors] = useState<Record<number, boolean>>({});
  const size = 10;
  const t = (key: Parameters<typeof translate>[1]) => translate(language, key);
  const page = parsePositiveInt(searchParams.get('page'), 1);
  const categoryId = parsePositiveInt(searchParams.get('categoryId'), 0);
  const tagId = parsePositiveInt(searchParams.get('tagId'), 0);
  const hasActiveFilters = Boolean(categoryId || tagId);

  useEffect(() => {
    const controller = new AbortController();
    const fetchArticles = async () => {
      setLoading(true);
      setError('');
      try {
        const res = await getArticles({
          status: 'published',
          page,
          size,
          ...(categoryId ? { categoryId } : {}),
          ...(tagId ? { tagId } : {}),
        }, controller.signal);
        if (controller.signal.aborted) {
          return;
        }
        setArticles(res.data.items || []);
        setTotal(res.data.total || 0);
        setCoverErrors({});
      } catch (err) {
        if (!controller.signal.aborted) {
          setError(getErrorMessage(err, translate(language, 'home.loadError')));
        }
      } finally {
        if (!controller.signal.aborted) {
          setLoading(false);
        }
      }
    };
    fetchArticles();
    return () => {
      controller.abort();
    };
  }, [page, categoryId, tagId, language]);

  const updateParams = (updates: Record<string, string | number | undefined>) => {
    const next = new URLSearchParams(searchParams);
    Object.entries(updates).forEach(([key, value]) => {
      if (value === undefined || value === '' || value === 0) {
        next.delete(key);
      } else {
        next.set(key, String(value));
      }
    });
    setSearchParams(next);
  };

  const handleClear = () => {
    // Home 路由不消费 q（仅 Search 路由使用），清空筛选只重置 categoryId/tagId/page。
    updateParams({ categoryId: undefined, tagId: undefined, page: undefined });
  };

  const handleCoverError = (articleId: number) => {
    setCoverErrors((current) => ({ ...current, [articleId]: true }));
  };

  const handleCoverLoad = (articleId: number) => {
    setCoverErrors((current) => {
      if (!current[articleId]) {
        return current;
      }
      const next = { ...current };
      delete next[articleId];
      return next;
    });
  };

  return (
    <div className="mx-auto mt-4 w-full max-w-4xl space-y-14 md:mt-8 md:space-y-20">
      {hasActiveFilters && (
        <div className="flex flex-col gap-3 border-b border-mountain-grey border-opacity-40 pb-6 text-sm text-ink-light md:flex-row md:items-center md:justify-between">
          <p className="tracking-widest opacity-75">
            {t('home.activeFilters')}
          </p>
          <Button
            variant="ghost"
            size="sm"
            onClick={handleClear}
            className="self-start md:self-auto"
          >
            {t('home.clearFiltersPoetic')}
          </Button>
        </div>
      )}
      <InlineNotice message={error} />
      {loading && articles.length === 0 && (
        <div className="space-y-14 md:space-y-20">
          {Array.from({ length: 5 }).map((_, index) => (
            <ArticleCardSkeleton key={index} />
          ))}
        </div>
      )}
      {loading && articles.length > 0 && (
        <PagePendingState
          variant="inline"
          label={t('common.loadingArticles')}
        />
      )}
      {!loading && articles.length === 0 ? (
        <EmptyState
          illustration="ink-drop"
          title={t('common.emptyArticles')}
          description={hasActiveFilters ? t('home.activeFilters') : undefined}
          action={hasActiveFilters ? { label: t('home.clearFiltersPoetic'), onClick: handleClear } : undefined}
        />
      ) : articles.length > 0 ? (
        <>
          {articles.map((article, index) => {
            const coverUrl = normalizeCoverUrl(article.coverUrl);
            const isCoverHidden = Boolean(coverUrl && coverErrors[article.id]);
            const shouldShowCover = Boolean(coverUrl && !isCoverHidden);
            const visibleCoverCountBefore = articles
              .slice(0, index)
              .filter((item) => {
                const itemCoverUrl = normalizeCoverUrl(item.coverUrl);
                return Boolean(itemCoverUrl && !coverErrors[item.id]);
              }).length;
            const shouldReverse = homeArticleLayout === 'alternating' && shouldShowCover && visibleCoverCountBefore % 2 === 1;

            return (
              <article key={article.id} className={`group relative flex flex-col gap-5 pb-2 items-start md:flex-row md:gap-8 md:pb-0 ${shouldReverse ? 'md:flex-row-reverse' : ''}`}>
                {shouldShowCover && (
                  <div className="relative aspect-[16/9] w-full shrink-0 overflow-hidden md:h-48 md:w-1/3 md:aspect-auto">
                    <PreloadLink to={`/article/${article.id}`} preload={routeLoaders.articleDetail} className="block h-full">
                      <img
                        src={coverUrl}
                        alt={article.title}
                        loading="lazy"
                        decoding="async"
                        onError={() => handleCoverError(article.id)}
                        onLoad={() => handleCoverLoad(article.id)}
                        className="w-full h-full object-cover grayscale opacity-80 group-hover:grayscale-0 group-hover:opacity-100 transition-[filter,opacity] duration-slow"
                      />
                    </PreloadLink>
                    <div className="absolute inset-0 bg-[var(--cover-wash-subtle)] pointer-events-none"></div>
                  </div>
                )}
                <div className="flex-1">
                  <PreloadLink to={`/article/${article.id}`} preload={routeLoaders.articleDetail} className="block">
                    <h2 className="mb-4 text-2xl font-bold leading-tight text-ink transition-colors duration-500 group-hover:text-ochre md:text-3xl">
                      {article.title}
                    </h2>
                    <p className="text-ink-light leading-relaxed mb-6 whitespace-pre-line line-clamp-3">
                      {article.summary}
                    </p>
                  </PreloadLink>
                  <div className="flex flex-wrap items-center gap-x-4 gap-y-2 text-xs text-ink-light opacity-70 tracking-wider">
                    {article.isPinned && (
                      <Tag tone="ochre" size="sm">{t('common.pinned')}</Tag>
                    )}
                    <span>{new Date(article.createdAt).toLocaleDateString(getDateLocale(language), { year: 'numeric', month: 'long', day: 'numeric' })}</span>
                    <span>{t('common.views')} {article.viewCount}</span>
                    {article.category && (
                      <Link to={`/?categoryId=${article.category.id}`} className="px-2 py-0.5 border border-mountain-grey border-opacity-50 hover:border-ochre hover:text-ochre transition-colors">
                        {article.category.name}
                      </Link>
                    )}
                    {article.tags && article.tags.length > 0 && (
                      <div className="flex flex-wrap gap-x-3 gap-y-2">
                        {article.tags.map(tg => (
                          <Link key={tg.id} to={`/?tagId=${tg.id}`} className="relative hover:text-ochre transition-colors before:content-['#'] before:mr-1 before:opacity-30">
                            {tg.name}
                          </Link>
                        ))}
                      </div>
                    )}
                  </div>
                </div>
                <div className="absolute -bottom-10 left-0 w-24 h-px bg-mountain-grey opacity-50 group-hover:w-full group-hover:bg-ochre transition-[width,background-color] duration-slow ease-paper"></div>
              </article>
            );
          })}

          <Pagination
            currentPage={page}
            total={total}
            pageSize={size}
            onPageChange={(nextPage) => updateParams({ page: nextPage === 1 ? undefined : nextPage })}
          />
        </>
      ) : null}
    </div>
  );
};

const parsePositiveInt = (value: string | null, fallback: number) => {
  if (!value) {
    return fallback;
  }
  const parsed = Number.parseInt(value, 10);
  return Number.isFinite(parsed) && parsed > 0 ? parsed : fallback;
};

export default Home;
