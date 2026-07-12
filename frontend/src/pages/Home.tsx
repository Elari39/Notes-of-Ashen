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
import { formatText, getDateLocale, translate } from '../i18n';
import { usePreferenceStore } from '../store/preferences';
import { getArticles } from '../api/article';
import { Article } from '../types';
import { normalizeCoverUrl } from '../utils/cover';
import { useSiteSettingsStore } from '../store/siteSettings';
import { routeLoaders } from '../routes/lazyRoutes';

const Home: React.FC = () => {
  const language = usePreferenceStore((state) => state.language);
  const homeArticleLayout = useSiteSettingsStore((state) => state.homeArticleLayout);
  const siteTitle = useSiteSettingsStore((state) => state.siteTitle);
  const siteDescription = useSiteSettingsStore((state) => state.siteDescription);
  const [searchParams, setSearchParams] = useSearchParams();
  const [articles, setArticles] = useState<Article[]>([]);
  const [loading, setLoading] = useState(true);
  const [total, setTotal] = useState(0);
  const [error, setError] = useState('');
  const [coverErrors, setCoverErrors] = useState<Record<number, boolean>>({});
  const [retryVersion, setRetryVersion] = useState(0);
  const size = 10;
  const t = (key: Parameters<typeof translate>[1]) => translate(language, key);
  const page = parsePositiveInt(searchParams.get('page'), 1);
  const categoryId = parsePositiveInt(searchParams.get('categoryId'), 0);
  const tagId = parsePositiveInt(searchParams.get('tagId'), 0);
  const hasActiveFilters = Boolean(categoryId || tagId);
  const latestArticle = articles[0];

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
  }, [page, categoryId, tagId, language, retryVersion]);

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

  const handleRetry = () => {
    setRetryVersion((version) => version + 1);
  };

  return (
    <div className="editorial-container w-full space-y-16 md:space-y-24">
      {!hasActiveFilters && (
        <section className="grid items-stretch gap-6 py-4 lg:grid-cols-[1.12fr_0.88fr] lg:gap-8 lg:py-8">
          <div className="flex flex-col justify-center py-8 lg:py-14">
            <p className="editorial-kicker">{t('home.heroKicker')}</p>
            <h1 className="mt-6 max-w-4xl font-display text-5xl leading-[0.98] tracking-[-0.035em] text-ink sm:text-6xl lg:text-7xl xl:text-[5.5rem]">
              {siteTitle}
            </h1>
            <p className="mt-7 max-w-2xl text-base leading-8 text-body md:text-lg">
              {siteDescription}
            </p>
            <div className="mt-9 flex flex-wrap gap-3">
              <a href="#latest-notes" className="inline-flex min-h-11 items-center rounded-md bg-ochre px-5 py-2.5 text-sm font-medium text-on-accent transition-[filter] hover:brightness-95">
                {t('home.heroBrowse')} <span aria-hidden="true" className="ml-2">↓</span>
              </a>
              <Link to="/search" className="inline-flex min-h-11 items-center rounded-md border border-hairline bg-paper px-5 py-2.5 text-sm font-medium text-ink transition-colors hover:border-ink">
                {t('nav.search')}
              </Link>
            </div>
          </div>

          <div className="editorial-dark-card relative min-h-[22rem] overflow-hidden lg:min-h-[30rem]">
            <div aria-hidden="true" className="absolute -right-16 -top-16 h-52 w-52 rounded-full border border-white/10"></div>
            <div aria-hidden="true" className="absolute -right-5 top-8 h-28 w-28 rounded-full bg-ochre opacity-90"></div>
            <div className="relative flex h-full flex-col justify-between gap-12">
              <div className="flex items-center justify-between gap-4 text-xs font-medium tracking-[0.18em] text-on-dark-soft">
                <span>{t('home.latestLabel')}</span>
                <span>{formatText(t('home.articleCount'), { total })}</span>
              </div>
              {latestArticle ? (
                <PreloadLink to={`/article/${latestArticle.id}`} preload={routeLoaders.articleDetail} className="group block">
                  <p className="text-xs text-on-dark-soft">{new Date(latestArticle.createdAt).toLocaleDateString(getDateLocale(language))}</p>
                  <h2 className="mt-4 max-w-xl font-display text-4xl leading-[1.05] text-on-dark transition-colors group-hover:text-ochre md:text-5xl">
                    {latestArticle.title}
                  </h2>
                  <span className="mt-8 inline-flex items-center text-sm font-medium text-on-dark">{t('home.heroBrowse')} <span className="ml-2">→</span></span>
                </PreloadLink>
              ) : (
                <div>
                  <p className="font-display text-4xl leading-tight text-on-dark">{t('common.emptyArticles')}</p>
                  <p className="mt-4 text-sm text-on-dark-soft">{siteDescription}</p>
                </div>
              )}
            </div>
          </div>
        </section>
      )}

      <section id="latest-notes" className="scroll-mt-24 space-y-10">
        <div className="flex items-end justify-between gap-6 border-b border-hairline pb-5">
          <div>
            <p className="editorial-kicker">{t('home.journalKicker')}</p>
            <h2 className="mt-3 editorial-section-title">{t('home.latestTitle')}</h2>
          </div>
          <p className="hidden text-sm text-muted sm:block">{formatText(t('home.articleCount'), { total })}</p>
        </div>
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
      <InlineNotice
        message={error}
        action={(
          <Button type="button" variant="ghost" size="sm" onClick={handleRetry}>
            {t('common.retry')}
          </Button>
        )}
      />
      {loading && articles.length === 0 && (
        <div className="grid gap-6 lg:grid-cols-2">
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
      {!loading && !error && articles.length === 0 ? (
        <EmptyState
          illustration="ink-drop"
          title={t('common.emptyArticles')}
          description={hasActiveFilters ? t('home.activeFilters') : undefined}
          action={hasActiveFilters ? { label: t('home.clearFiltersPoetic'), onClick: handleClear } : undefined}
        />
      ) : articles.length > 0 ? (
        <>
          <div className={homeArticleLayout === 'alternating' ? 'space-y-6' : 'grid gap-6 md:grid-cols-2'}>
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
              <article key={article.id} className={`group relative overflow-hidden rounded-lg bg-surface-card ${homeArticleLayout === 'alternating' ? `flex flex-col items-stretch md:min-h-64 md:flex-row ${shouldReverse ? 'md:flex-row-reverse' : ''}` : 'flex flex-col'}`}>
                {shouldShowCover && (
                  <div className={`relative aspect-[16/9] w-full shrink-0 overflow-hidden ${homeArticleLayout === 'alternating' ? 'md:h-auto md:w-[42%] md:aspect-auto' : ''}`}>
                    <PreloadLink to={`/article/${article.id}`} preload={routeLoaders.articleDetail} className="block h-full">
                      <img
                        src={coverUrl}
                        alt={article.title}
                        loading="lazy"
                        decoding="async"
                        onError={() => handleCoverError(article.id)}
                        onLoad={() => handleCoverLoad(article.id)}
                        className="h-full w-full object-cover opacity-90 transition-[opacity,transform] duration-slow group-hover:scale-[1.015] group-hover:opacity-100"
                      />
                    </PreloadLink>
                    <div className="absolute inset-0 bg-[var(--cover-wash-subtle)] pointer-events-none"></div>
                  </div>
                )}
                <div className="flex flex-1 flex-col justify-between p-6 md:p-8">
                  <PreloadLink to={`/article/${article.id}`} preload={routeLoaders.articleDetail} className="block">
                    <h2 className="mb-4 font-display text-3xl leading-[1.08] text-ink transition-colors duration-base group-hover:text-ochre md:text-4xl">
                      {article.title}
                    </h2>
                    <p className="mb-7 line-clamp-3 whitespace-pre-line text-sm leading-7 text-body">
                      {article.summary}
                    </p>
                  </PreloadLink>
                  <div className="flex flex-wrap items-center gap-x-4 gap-y-2 text-xs text-muted">
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
              </article>
            );
          })}
          </div>

          <Pagination
            currentPage={page}
            total={total}
            pageSize={size}
            onPageChange={(nextPage) => updateParams({ page: nextPage === 1 ? undefined : nextPage })}
          />
        </>
      ) : null}
      </section>
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
