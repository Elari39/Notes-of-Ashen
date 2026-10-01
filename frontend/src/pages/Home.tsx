import React, { useEffect, useState } from 'react';
import { useSearchParams } from 'react-router-dom';
import Pagination from '../components/Pagination';
import InlineNotice from '../components/InlineNotice';
import PagePendingState from '../components/RoutePending';
import ArticleCardSkeleton from '../components/ArticleCardSkeleton';
import EmptyState from '../components/ui/EmptyState';
import Button from '../components/ui/Button';
import HomeArticleCard from '../components/HomeArticleCard';
import { PreloadLink } from '../components/PreloadLink';
import { getErrorMessage } from '../utils/error';
import { formatArticleCount, getDateLocale, translate } from '../i18n';
import { usePreferenceStore } from '../store/preferences';
import { getArticles } from '../api/article';
import { Article } from '../types';
import { normalizeCoverUrl } from '../utils/cover';
import { selectHomeArticles } from '../utils/homeArticles';
import {
  DEFAULT_SITE_DESCRIPTION,
  DEFAULT_SITE_TITLE,
  useSiteSettingsStore,
} from '../store/siteSettings';
import { routeLoaders } from '../routes/lazyRoutes';

const Home: React.FC = () => {
  const language = usePreferenceStore((state) => state.language);
  const homeArticleLayout = useSiteSettingsStore((state) => state.homeArticleLayout);
  const homeCtaHidden = useSiteSettingsStore((state) => state.homeCtaHidden);
  const siteTitle = useSiteSettingsStore((state) => state.siteTitle);
  const siteDescription = useSiteSettingsStore((state) => state.siteDescription);
  const [searchParams, setSearchParams] = useSearchParams();
  const [articles, setArticles] = useState<Article[]>([]);
  const [loading, setLoading] = useState(true);
  const [total, setTotal] = useState(0);
  const [error, setError] = useState('');
  const [retryVersion, setRetryVersion] = useState(0);
  const [coverErrors, setCoverErrors] = useState<Record<number, boolean>>({});
  const size = 10;
  const t = (key: Parameters<typeof translate>[1]) => translate(language, key);
  const page = parsePositiveInt(searchParams.get('page'), 1);
  const categoryId = parsePositiveInt(searchParams.get('categoryId'), 0);
  const tagId = parsePositiveInt(searchParams.get('tagId'), 0);
  const hasActiveFilters = Boolean(categoryId || tagId);
  const shouldShowHero = page === 1 && !hasActiveFilters;
  const { featuredArticle, listArticles } = selectHomeArticles(articles, shouldShowHero);
  const NotesHeading = shouldShowHero ? 'h2' : 'h1';
  const normalizedSiteTitle = siteTitle.trim();
  const normalizedSiteDescription = siteDescription.trim();
  const displaySiteTitle = !normalizedSiteTitle || normalizedSiteTitle === DEFAULT_SITE_TITLE
    ? t('brand.name')
    : normalizedSiteTitle;
  const displaySiteDescription = !normalizedSiteDescription || normalizedSiteDescription === DEFAULT_SITE_DESCRIPTION
    ? t('home.defaultSiteDescription')
    : normalizedSiteDescription;
  const articleCountLabel = formatArticleCount(language, total);
  const shouldShowLatestSection = !shouldShowHero || listArticles.length > 0 || total > size;
  const heroPanelHeight = loading || featuredArticle
    ? 'min-h-[16rem]'
    : 'min-h-[12rem]';

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
        setCoverErrors({});
        setTotal(res.data.total || 0);
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

  const handleRetry = () => {
    setRetryVersion((version) => version + 1);
  };

  return (
    <div className="home-page editorial-container w-full space-y-10 md:space-y-12">
      {shouldShowHero && (
        <section className="home-hero grid gap-6" aria-labelledby="home-hero-title">
          <div className="home-hero-intro min-w-0 py-2">
            <p className="editorial-kicker">{t('home.heroKicker')}</p>
            <h1 id="home-hero-title" className="mt-5 max-w-3xl break-words font-display text-4xl font-normal leading-[1.04] tracking-[-0.03em] text-ink [text-wrap:balance] sm:mt-6 sm:text-5xl lg:text-[4rem]">
              {displaySiteTitle}
            </h1>
            <p className="mt-5 max-w-xl text-base leading-8 text-body sm:mt-7 md:text-lg">
              {displaySiteDescription}
            </p>
            <div className="mt-7 flex flex-wrap gap-3 sm:mt-9">
              {listArticles.length > 0 && (
                <a
                  href="#latest-notes"
                  className="theme-action inline-flex min-h-11 items-center rounded-md bg-ochre px-5 py-2.5 text-sm font-medium text-on-accent transition-[filter] hover:brightness-95 focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-ochre"
                >
                  {t('home.heroBrowse')} <span aria-hidden="true" className="ml-2">↓</span>
                </a>
              )}
              <PreloadLink
                to="/search"
                preload={routeLoaders.search}
                className="theme-action inline-flex min-h-11 items-center rounded-md border border-hairline bg-paper px-5 py-2.5 text-sm font-medium text-ink transition-colors hover:border-ink focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-ochre"
              >
                {t('nav.search')}
              </PreloadLink>
            </div>
          </div>

          <div className={`home-feature relative overflow-hidden rounded-xl bg-surface-dark p-6 text-on-dark shadow-sm transition-[min-height] duration-slow sm:p-7 md:p-9 ${heroPanelHeight}`}>
            <div className="flex h-full flex-col">
              <div className="home-feature-meta flex items-start justify-between gap-3 border-b border-white/10 pb-5 text-xs font-medium tracking-[0.16em] text-on-dark-soft sm:gap-4 sm:tracking-[0.18em]">
                <span className="max-w-[9rem] leading-5 sm:max-w-none">{t('home.featuredLabel')}</span>
                {loading && articles.length === 0 ? (
                  <span aria-label={t('common.loadingArticles')} className="text-right">…</span>
                ) : (
                  <span className="max-w-[9rem] text-right leading-5 sm:max-w-none">{articleCountLabel}</span>
                )}
              </div>

              {loading && !featuredArticle ? (
                <div className="mt-auto space-y-4 pt-10 sm:space-y-5 sm:pt-6" aria-hidden="true">
                  <div className="home-feature-skeleton h-3 w-28 animate-pulse rounded-full bg-white/10"></div>
                  <div className="home-feature-skeleton h-12 w-full animate-pulse rounded-md bg-white/10"></div>
                  <div className="home-feature-skeleton h-12 w-4/5 animate-pulse rounded-md bg-white/10"></div>
                  <div className="home-feature-skeleton h-4 w-2/3 animate-pulse rounded-full bg-white/10"></div>
                </div>
              ) : error && !featuredArticle ? (
                <div className="mt-auto pt-10" role="alert">
                  <p className="editorial-kicker text-ochre">{t('home.loadErrorTitle')}</p>
                  <p className="mt-4 max-w-md font-display text-3xl leading-tight text-on-dark sm:text-4xl">
                    {error}
                  </p>
                  <button
                    type="button"
                    onClick={handleRetry}
                    className="mt-6 inline-flex min-h-11 items-center rounded-md border border-white/20 px-5 py-2.5 text-sm font-medium text-on-dark transition-colors hover:border-ochre hover:text-ochre focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-ochre"
                  >
                    {t('common.retry')}
                  </button>
                </div>
              ) : featuredArticle ? (
                <PreloadLink
                  to={`/article/${featuredArticle.id}`}
                  preload={routeLoaders.articleDetail}
                  className="home-feature-link group flex min-w-0 flex-1 flex-col justify-end pt-5 focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-4 focus-visible:outline-ochre sm:pt-6"
                >
                  <div className="mb-5 flex flex-wrap items-center gap-3 text-xs text-on-dark-soft">
                    {featuredArticle.isPinned && (
                      <span className="rounded-full bg-ochre px-3 py-1 font-medium text-on-accent">{t('common.pinned')}</span>
                    )}
                    {featuredArticle.category && <span>{featuredArticle.category.name}</span>}
                  </div>
                  <p className="text-xs text-on-dark-soft">
                    {new Date(featuredArticle.publishedAt || featuredArticle.createdAt).toLocaleDateString(getDateLocale(language), { year: 'numeric', month: 'long', day: 'numeric' })}
                  </p>
                  <h2 className="mt-4 max-w-xl break-words font-display text-3xl font-normal leading-[1.08] tracking-[-0.025em] text-on-dark transition-colors group-hover:text-ochre [text-wrap:balance] sm:text-4xl md:text-5xl">
                    {featuredArticle.title}
                  </h2>
                  {featuredArticle.summary && (
                    <p className="mt-5 line-clamp-3 max-w-xl text-sm leading-7 text-on-dark-soft">
                      {featuredArticle.summary}
                    </p>
                  )}
                  <span className="mt-8 inline-flex min-h-11 items-center text-sm font-medium text-on-dark">
                    {t('home.featuredRead')} <span aria-hidden="true" className="ml-2">→</span>
                  </span>
                </PreloadLink>
              ) : (
                <div className="mt-auto pt-10" aria-live="polite">
                  <p className="editorial-kicker text-ochre">{t('home.emptyKicker')}</p>
                  <p className="mt-4 font-display text-3xl leading-tight text-on-dark sm:text-4xl">{t('home.emptyTitle')}</p>
                  <p className="mt-4 max-w-md text-sm leading-7 text-on-dark-soft">{t('home.emptyDescription')}</p>
                </div>
              )}
            </div>
          </div>
        </section>
      )}

      {shouldShowLatestSection && (
      <section id="latest-notes" className="scroll-mt-24 space-y-8 sm:space-y-10" aria-labelledby="latest-notes-title">
        <div className="home-section-heading flex flex-col items-start justify-between gap-3 border-b border-hairline pb-5 sm:flex-row sm:items-end sm:gap-6">
          <div>
            <p className="editorial-kicker">{t('home.journalKicker')}</p>
            <NotesHeading id="latest-notes-title" className="mt-3 editorial-section-title">{t('home.latestTitle')}</NotesHeading>
          </div>
          <p className="text-xs leading-5 text-muted sm:max-w-[12rem] sm:text-right sm:text-sm">{articleCountLabel}</p>
        </div>

        {hasActiveFilters && (
          <div className="flex flex-col gap-3 rounded-lg bg-surface-soft px-5 py-4 text-sm text-ink-light md:flex-row md:items-center md:justify-between">
            <p className="tracking-widest opacity-75">{t('home.activeFilters')}</p>
            <Button variant="ghost" size="sm" onClick={handleClear} className="self-start md:self-auto">
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
          <div className={homeArticleLayout === 'alternating' ? 'space-y-6' : 'grid gap-5 md:grid-cols-2 lg:gap-6 xl:grid-cols-3'}>
            {Array.from({ length: homeArticleLayout === 'alternating' ? 3 : 6 }).map((_, index) => (
              <ArticleCardSkeleton key={index} variant={homeArticleLayout} />
            ))}
          </div>
        )}

        {loading && articles.length > 0 && (
          <PagePendingState variant="inline" label={t('common.loadingArticles')} />
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
            {listArticles.length > 0 && (
              <div className={homeArticleLayout === 'alternating' ? 'space-y-6' : 'grid gap-5 md:grid-cols-2 lg:gap-6 xl:grid-cols-3'}>
                {listArticles.map((article, index) => (
                  <HomeArticleCard key={`${article.id}:${article.coverUrl}`} article={article} layout={homeArticleLayout}
                    coverFailed={Boolean(coverErrors[article.id])}
                    reverseCover={listArticles.slice(0, index).filter((item) => normalizeCoverUrl(item.coverUrl) && !coverErrors[item.id]).length % 2 === 1}
                    onCoverError={(id) => setCoverErrors((current) => ({ ...current, [id]: true }))}
                    position={(page - 1) * size + articles.findIndex((item) => item.id === article.id) + 1} />
                ))}
              </div>
            )}

            <Pagination
              currentPage={page}
              total={total}
              pageSize={size}
              onPageChange={(nextPage) => updateParams({ page: nextPage === 1 ? undefined : nextPage })}
            />
          </>
        ) : null}
      </section>
      )}

      {!homeCtaHidden && !loading && !error && articles.length > 0 && (
        <section aria-labelledby="home-cta-title" className="home-cta rounded-lg bg-ochre px-6 py-10 text-on-accent md:px-12 md:py-14 lg:px-16 lg:py-16">
          <div className="grid gap-8 lg:grid-cols-[1fr_auto] lg:items-end">
            <div>
              <p className="text-xs font-medium tracking-[0.18em]">{t('home.ctaKicker')}</p>
              <h2 id="home-cta-title" className="mt-4 max-w-3xl font-display text-3xl font-normal leading-tight tracking-[-0.02em] md:text-4xl">
                {t('home.ctaTitle')}
              </h2>
              <p className="mt-4 max-w-2xl text-sm leading-7 opacity-80 md:text-base">{t('home.ctaDescription')}</p>
            </div>
            <div className="flex flex-wrap gap-3">
              <PreloadLink
                to="/archive"
                preload={routeLoaders.archive}
                className="inline-flex min-h-11 items-center rounded-md bg-paper px-5 py-2.5 text-sm font-medium text-ink transition-[filter] hover:brightness-95 focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-ink"
              >
                {t('home.ctaArchive')}
              </PreloadLink>
              <PreloadLink
                to="/search"
                preload={routeLoaders.search}
                className="inline-flex min-h-11 items-center rounded-md border border-current px-5 py-2.5 text-sm font-medium transition-colors hover:bg-black/5 focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-current"
              >
                {t('home.ctaSearch')}
              </PreloadLink>
            </div>
          </div>
        </section>
      )}
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
