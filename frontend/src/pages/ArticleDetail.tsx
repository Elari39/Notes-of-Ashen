import React, { useCallback, useEffect, useMemo, useRef, useState } from 'react';
import { Link, useParams } from 'react-router-dom';
import ImageLightbox, { LightboxImage } from '../components/ImageLightbox';
import InlineNotice from '../components/InlineNotice';
import MarkdownRenderer from '../components/MarkdownRenderer';
import PagePendingState, { RoutePendingIndicator } from '../components/RoutePending';
import ArticleDetailSkeleton from '../components/ArticleDetailSkeleton';
import { PreloadLink } from '../components/PreloadLink';
import Button from '../components/ui/Button';
import { getArticleById, getArticleContext, likeArticle } from '../api/article';
import { useSEO } from '../utils/seo';
import { getErrorMessage, toAppError } from '../utils/error';
import { formatText, getDateLocale, translate } from '../i18n';
import { usePreferenceStore, type Language } from '../store/preferences';
import { useReadingProgress } from '../hooks/useReadingProgress';
import { normalizeCoverUrl } from '../utils/cover';
import { extractMarkdownHeadings, type MarkdownHeading } from '../utils/markdownHeadings';
import type { Article, ArticleContext } from '../types';
import { routeLoaders } from '../routes/lazyRoutes';
import { safeLocalStorage } from '../utils/storage';
import { useSiteSettingsStore } from '../store/siteSettings';
import { toast } from '../utils/notify';

type ArticleDetailData = Article & { content: string };

const ArticleDetail: React.FC = () => {
  const { id } = useParams<{ id: string }>();
  const language = usePreferenceStore((state) => state.language);
  const siteBaseUrl = useSiteSettingsStore((state) => state.siteBaseUrl);
  const [article, setArticle] = useState<ArticleDetailData | null>(null);
  const [articleContext, setArticleContext] = useState<ArticleContext | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');
  const [isNotFound, setIsNotFound] = useState(false);
  const [retryNonce, setRetryNonce] = useState(0);
  const [contextError, setContextError] = useState('');
  const [likeError, setLikeError] = useState('');
  const [likeCount, setLikeCount] = useState(0);
  const [hasLiked, setHasLiked] = useState(false);
  const [isLiking, setIsLiking] = useState(false);
  const [activeHeadingId, setActiveHeadingId] = useState('');
  const [tocCollapsed, setTocCollapsed] = useState(false);
  const readingProgressRef = useReadingProgress();
  const [coverError, setCoverError] = useState(false);
  const [lightboxImage, setLightboxImage] = useState<LightboxImage | null>(null);
  const closeLightbox = useCallback(() => setLightboxImage(null), []);
  const t = (key: Parameters<typeof translate>[1]) => translate(language, key);
  // 用 ref 持有最新 language，避免切换语言时重新拉取文章正文（正文与语言无关，
  // language 仅用于错误兜底文案）。
  const languageRef = useRef(language);
  languageRef.current = language;

  useSEO(article?.seoTitle || article?.title, article?.seoDescription || article?.summary, article?.seoKeywords);

  const headings = useMemo(() => extractMarkdownHeadings(article?.content || '', 3), [article?.content]);

  useEffect(() => {
    const controller = new AbortController();
    const fetchArticle = async () => {
      if (!id) {
        setArticle(null);
        setArticleContext(null);
        setIsNotFound(true);
        setLoading(false);
        return;
      }

      setLoading(true);
      setError('');
      setIsNotFound(false);
      setContextError('');
      setLikeError('');
      setArticle(null);
      setArticleContext(null);
      setActiveHeadingId('');
      setCoverError(false);
      setLightboxImage(null);

      try {
        const res = await getArticleById(id, controller.signal);
        if (controller.signal.aborted) {
          return;
        }
        setArticle({ ...res.data, content: res.data.content || '' });
        setLikeCount(res.data.likeCount || 0);
        setHasLiked(safeLocalStorage.getItem(articleLikeStorageKey(res.data.id)) === '1');

        try {
          const contextRes = await getArticleContext(id, controller.signal);
          if (!controller.signal.aborted) {
            setArticleContext(contextRes.data);
          }
        } catch (contextErr) {
          if (!controller.signal.aborted) {
            setContextError(getErrorMessage(contextErr, translate(languageRef.current, 'article.loadError')));
          }
        }
      } catch (err) {
        if (!controller.signal.aborted) {
          const appError = toAppError(err, translate(languageRef.current, 'article.loadError'));
          setError(appError.message);
          setIsNotFound(appError.status === 404);
        }
      } finally {
        if (!controller.signal.aborted) {
          setLoading(false);
        }
      }
    };

    fetchArticle();
    return () => {
      controller.abort();
    };
  }, [id, retryNonce]);

  const retryArticle = () => {
    setRetryNonce((value) => value + 1);
  };

  useEffect(() => {
    const fallbackActiveHeading = () => {
      if (headings.length === 0) {
        setActiveHeadingId('');
        return;
      }

      let active = '';
      for (const heading of headings) {
        const element = document.getElementById(heading.id);
        if (!element) {
          continue;
        }
        if (element.getBoundingClientRect().top <= 128) {
          active = heading.id;
        }
      }
      setActiveHeadingId(active);
    };

    if (headings.length === 0 || typeof IntersectionObserver === 'undefined') {
      fallbackActiveHeading();
      window.addEventListener('scroll', fallbackActiveHeading, { passive: true });
      window.addEventListener('resize', fallbackActiveHeading);
      return () => {
        window.removeEventListener('scroll', fallbackActiveHeading);
        window.removeEventListener('resize', fallbackActiveHeading);
      };
    }

    const visible = new Set<string>();
    const observer = new IntersectionObserver(
      (entries) => {
        entries.forEach((entry) => {
          const id = entry.target.id;
          if (entry.isIntersecting) {
            visible.add(id);
          } else {
            visible.delete(id);
          }
        });
        const next = headings.find((heading) => visible.has(heading.id));
        if (next) {
          setActiveHeadingId(next.id);
        } else {
          fallbackActiveHeading();
        }
      },
      {
        rootMargin: '-112px 0px -62% 0px',
        threshold: [0, 0.2, 1],
      },
    );

    headings.forEach((heading) => {
      const element = document.getElementById(heading.id);
      if (element) {
        observer.observe(element);
      }
    });
    fallbackActiveHeading();

    return () => {
      observer.disconnect();
    };
  }, [headings]);

  const handleLike = async () => {
    if (!article || hasLiked || isLiking) {
      return;
    }
    setLikeError('');
    setIsLiking(true);
    try {
      const res = await likeArticle(article.id);
      setLikeCount(res.data.likeCount);
      // 以后端返回的 liked 为准，localStorage 仅作跨会话辅助记忆。
      const liked = res.data.liked ?? true;
      setHasLiked(liked);
      if (liked) {
        safeLocalStorage.setItem(articleLikeStorageKey(article.id), '1');
      }
    } catch (err) {
      setLikeError(getErrorMessage(err, t('articleDetail.likeError')));
    } finally {
      setIsLiking(false);
    }
  };

  const shareURL = getArticleShareURL(article?.id, siteBaseUrl);
  const handleShare = async () => {
    if (!article) return;
    if (navigator.share) {
      try {
        await navigator.share({ title: article.title, text: article.summary, url: shareURL });
      } catch (err) {
        if (!(err instanceof DOMException) || err.name !== 'AbortError') {
          toast.error('share.failed');
        }
      }
      return;
    }
    await copyArticleLink(shareURL);
  };

  if (loading && !article) {
    return <ArticleDetailSkeleton />;
  }

  if (!article) {
    const message = error || t('article.missing');
    return (
      <div className="mx-auto mt-20 max-w-md space-y-5">
        <InlineNotice
          message={message}
          action={!isNotFound && error ? (
            <Button size="sm" onClick={retryArticle}>
              {t('common.retry')}
            </Button>
          ) : undefined}
        />
        <Link to="/" className="inline-block border border-ink px-5 py-2 text-sm tracking-widest text-ink transition-colors hover:bg-ink hover:text-paper">
          {t('common.backHome')}
        </Link>
      </div>
    );
  }

  const coverUrl = normalizeCoverUrl(article.coverUrl);
  const isCoverHidden = Boolean(coverUrl && coverError);

  return (
    <>
      {loading && <RoutePendingIndicator />}
      <div
        ref={readingProgressRef}
        className="fixed left-0 top-0 z-[70] h-px bg-ochre transition-[width] duration-150"
        style={{ width: '0%' }}
        aria-hidden="true"
      />

      <div className="editorial-container grid w-full gap-10 lg:grid-cols-[minmax(0,48rem)_16rem] lg:items-start lg:justify-center xl:gap-16">

        <article className="min-w-0 w-full">
          {loading && <PagePendingState variant="inline" label={t('common.loadingArticle')} />}
          <InlineNotice message={error} className="mb-6" />
          <header className="mb-14">
            {coverUrl && (
              <div className={`relative mb-12 w-full overflow-hidden rounded-xl ${isCoverHidden ? 'h-28 md:h-32' : 'h-72 md:h-[28rem]'}`}>
                {isCoverHidden ? (
                  <div className="flex h-full items-center justify-center border border-mountain-grey bg-[var(--paper-soft)] text-xs tracking-widest text-ink-light opacity-70">
                    {t('article.coverHidden')}
                  </div>
                ) : (
                  <>
                    <button
                      type="button"
                      className="block h-full w-full cursor-zoom-in bg-transparent p-0"
                      aria-label={formatText(t('articleDetail.viewImage'), { title: article.title })}
                      onClick={() => setLightboxImage({ src: coverUrl, alt: article.title })}
                    >
                      <img
                        src={coverUrl}
                        alt={article.title}
                        loading="eager"
                        fetchPriority="high"
                        decoding="async"
                        onError={() => setCoverError(true)}
                        onLoad={() => setCoverError(false)}
                        className="h-full w-full object-cover opacity-95 transition-[opacity,transform] duration-slow hover:scale-[1.01] hover:opacity-100"
                      />
                    </button>
                    <div className="absolute inset-0 bg-[var(--cover-wash)] pointer-events-none" />
                  </>
                )}
              </div>
            )}

            <p className="editorial-kicker mb-5">{t('articleDetail.kicker')}</p>
            <h1 className="mb-7 font-display text-5xl leading-[1.02] tracking-[-0.035em] text-ink md:text-6xl">
              {article.title}
            </h1>

            <div className="flex flex-col space-y-4 text-sm text-muted">
              <div className="flex flex-wrap gap-x-6 gap-y-2">
                <span>{new Date(article.createdAt).toLocaleDateString(getDateLocale(language), { year: 'numeric', month: 'long', day: 'numeric' })}</span>
                <span>{t('common.views')} {article.viewCount}</span>
                <span>{t('articleDetail.likes')} {likeCount}</span>
                <span>{formatText(t('reading.words'), { count: article.wordCount })}</span>
                <span>{formatText(t('reading.minutes'), { count: article.readingTimeMinutes })}</span>
              </div>

              <div className="flex flex-wrap items-center gap-3 text-xs">
                {article.category && (
                  <Link to={`/?categoryId=${article.category.id}`} className="rounded-full bg-surface-card px-3 py-1.5 font-medium text-ink transition-colors hover:bg-surface-strong">
                    {article.category.name}
                  </Link>
                )}
                {article.tags && article.tags.length > 0 && (
                  <div className="flex flex-wrap justify-center gap-3">
                    {article.tags.map((tag) => (
                      <Link key={tag.id} to={`/?tagId=${tag.id}`} className="relative hover:text-ochre transition-colors before:content-['#'] before:mr-1 before:opacity-30">
                        {tag.name}
                      </Link>
                    ))}
                  </div>
                )}
              </div>
            </div>
          </header>

          {headings.length > 0 && (
            <ArticleTOC
              headings={headings}
              activeHeadingId={activeHeadingId}
              collapsed={tocCollapsed}
              onToggle={() => setTocCollapsed((value) => !value)}
              language={language}
              className="mb-10 lg:hidden"
            />
          )}

          <MarkdownRenderer content={article.content} headings={headings} className="prose-lg mx-auto" />

          <div className="mt-16 rounded-xl bg-surface-soft px-6 py-8 text-center">
            <button
              type="button"
              onClick={handleLike}
              disabled={hasLiked || isLiking}
              className={`group inline-flex min-h-11 items-center gap-3 rounded-md border px-5 py-2 text-sm font-medium transition-colors ${
                hasLiked
                  ? 'border-ochre bg-ochre text-on-accent'
                  : 'border-hairline bg-paper text-ink hover:border-ink'
              } disabled:cursor-not-allowed disabled:opacity-80`}
              aria-pressed={hasLiked}
            >
              <span className={`h-2.5 w-2.5 rounded-full ${hasLiked ? 'bg-[var(--on-accent)]' : 'bg-ochre group-hover:scale-125'} transition-transform`} />
              <span>{hasLiked ? t('articleDetail.liked') : (isLiking ? t('articleDetail.liking') : t('articleDetail.like'))}</span>
              <span>{likeCount}</span>
            </button>
            <InlineNotice message={likeError} className="mt-4" />
            <div className="mt-5 flex flex-wrap justify-center gap-3">
              {typeof navigator.share === 'function' && (
                <Button type="button" size="sm" variant="ghost" onClick={() => void handleShare()}>
                  {t('share.native')}
                </Button>
              )}
              <Button type="button" size="sm" variant="ghost" onClick={() => void copyArticleLink(shareURL)}>
                {t('share.copy')}
              </Button>
            </div>
          </div>

          <div className="mt-10 text-center">
            <Link to="/" className="inline-flex min-h-11 items-center rounded-md border border-hairline bg-paper px-6 py-2 text-sm font-medium text-ink transition-colors hover:border-ink">
              {t('common.backHome')}
            </Link>
          </div>

          <ArticleContextBlock context={articleContext} error={contextError} language={language} />
          <ImageLightbox image={lightboxImage} onClose={closeLightbox} />
        </article>

        {headings.length > 0 && (
          <ArticleTOC
            headings={headings}
            activeHeadingId={activeHeadingId}
            collapsed={tocCollapsed}
            onToggle={() => setTocCollapsed((value) => !value)}
            language={language}
            className="sticky top-24 hidden lg:block"
          />
        )}
      </div>
    </>
  );
};

const copyArticleLink = async (url: string) => {
  try {
    if (navigator.clipboard?.writeText) {
      await navigator.clipboard.writeText(url);
    } else {
      const input = document.createElement('textarea');
      input.value = url;
      input.style.position = 'fixed';
      input.style.opacity = '0';
      document.body.appendChild(input);
      input.select();
      const copied = document.execCommand('copy');
      input.remove();
      if (!copied) throw new Error('copy failed');
    }
    toast.success('share.copied');
  } catch {
    toast.error('share.failed');
  }
};

const getArticleShareURL = (articleID: number | undefined, siteBaseUrl: string) => {
  if (!articleID) return window.location.href;
  try {
    return new URL(`/article/${articleID}`, siteBaseUrl.trim() || window.location.origin).toString();
  } catch {
    return new URL(`/article/${articleID}`, window.location.origin).toString();
  }
};

const ArticleTOC: React.FC<{
  headings: MarkdownHeading[];
  activeHeadingId: string;
  collapsed: boolean;
  onToggle: () => void;
  language: Language;
  className?: string;
}> = ({ headings, activeHeadingId, collapsed, onToggle, language, className = '' }) => {
  if (headings.length === 0) {
    return null;
  }

  const t = (key: Parameters<typeof translate>[1]) => translate(language, key);

  return (
    <aside className={`rounded-lg bg-surface-soft p-5 ${className}`.trim()} aria-label={t('articleToc.title')}>
      <div className="mb-3 flex items-center justify-between gap-3">
        <h2 className="font-display text-xl text-ink">{t('articleToc.title')}</h2>
        <button
          type="button"
          onClick={onToggle}
          className="inline-flex min-h-11 items-center rounded-md border border-hairline bg-paper px-3 py-2 text-xs text-muted transition-colors hover:border-ink hover:text-ink"
          aria-expanded={!collapsed}
        >
          {collapsed ? t('articleToc.expand') : t('articleToc.collapse')}
        </button>
      </div>

      {!collapsed && (
        <nav className="max-h-[calc(100vh-8rem)] space-y-2 overflow-y-auto pr-1 text-sm">
          {headings.map((heading) => (
            <button
              key={heading.id}
              type="button"
              onClick={() => document.getElementById(heading.id)?.scrollIntoView({ behavior: 'smooth', block: 'start' })}
              className={`block w-full truncate border-l px-3 py-1.5 text-left transition-colors ${
                activeHeadingId === heading.id
                  ? 'border-ochre text-ink'
                  : 'border-hairline text-muted hover:border-ochre hover:text-ink'
              }`}
              style={{ paddingLeft: `${Math.max(0, heading.depth - 1) * 0.65 + 0.75}rem` }}
              title={heading.title}
            >
              {heading.title}
            </button>
          ))}
        </nav>
      )}
    </aside>
  );
};

const ArticleContextBlock: React.FC<{ context: ArticleContext | null; error: string; language: Language }> = ({ context, error, language }) => {
  const hasNavigation = Boolean(context?.previous || context?.next);
  const hasRelated = Boolean(context?.related?.length);
  const t = (key: Parameters<typeof translate>[1]) => translate(language, key);

  if (error) {
    return <InlineNotice message={error} className="mt-10" />;
  }

  if (!context || (!hasNavigation && !hasRelated)) {
    return null;
  }

  return (
    <section className="mt-16 border-t border-hairline pt-12">
      {hasNavigation && (
        <div className="grid gap-4 md:grid-cols-2">
          <ArticleNavLink label={t('articleDetail.previous')} article={context.previous} align="left" />
          <ArticleNavLink label={t('articleDetail.next')} article={context.next} align="right" />
        </div>
      )}

      {hasRelated && (
        <div className="mt-12">
          <h2 className="mb-5 font-display text-3xl text-ink">{t('articleDetail.related')}</h2>
          <div className="grid gap-4 md:grid-cols-3">
            {context.related.map((item) => (
              <PreloadLink
                key={item.id}
                to={`/article/${item.id}`}
                preload={routeLoaders.articleDetail}
                className="rounded-lg bg-surface-card p-5 transition-colors hover:bg-surface-strong"
              >
                <h3 className="line-clamp-2 font-display text-2xl leading-tight text-ink">{item.title}</h3>
                <p className="mt-3 line-clamp-3 text-sm leading-relaxed text-ink-light">{item.summary}</p>
              </PreloadLink>
            ))}
          </div>
        </div>
      )}
    </section>
  );
};

const ArticleNavLink: React.FC<{
  label: string;
  article?: Article;
  align: 'left' | 'right';
}> = ({ label, article, align }) => {
  if (!article) {
    return <div className="hidden md:block" />;
  }

  return (
    <PreloadLink
      to={`/article/${article.id}`}
      preload={routeLoaders.articleDetail}
      className={`block rounded-lg bg-surface-card p-5 transition-colors hover:bg-surface-strong ${align === 'right' ? 'md:text-right' : ''}`}
    >
      <span className="text-xs tracking-widest text-ink-light">{label}</span>
      <h2 className="mt-2 line-clamp-2 font-display text-2xl leading-tight text-ink">{article.title}</h2>
    </PreloadLink>
  );
};

const articleLikeStorageKey = (articleID: number) => `article-like:${articleID}`;

export default ArticleDetail;
