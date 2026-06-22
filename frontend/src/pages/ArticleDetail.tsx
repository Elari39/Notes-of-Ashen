import React, { useCallback, useEffect, useMemo, useRef, useState } from 'react';
import { Link, useParams } from 'react-router-dom';
import ImageLightbox, { LightboxImage } from '../components/ImageLightbox';
import InlineNotice from '../components/InlineNotice';
import MarkdownRenderer from '../components/MarkdownRenderer';
import PagePendingState, { RoutePendingIndicator } from '../components/RoutePending';
import ArticleDetailSkeleton from '../components/ArticleDetailSkeleton';
import { PreloadLink } from '../components/PreloadLink';
import { getArticleById, getArticleContext, likeArticle } from '../api/article';
import { useSEO } from '../utils/seo';
import { getErrorMessage } from '../utils/error';
import { getDateLocale, translate } from '../i18n';
import { usePreferenceStore } from '../store/preferences';
import { useReadingProgress } from '../hooks/useReadingProgress';
import { normalizeCoverUrl } from '../utils/cover';
import { extractMarkdownHeadings, type MarkdownHeading } from '../utils/markdownHeadings';
import type { Article, ArticleContext } from '../types';
import { routeLoaders } from '../routes/lazyRoutes';

type ArticleDetailData = Article & { content: string };

const ArticleDetail: React.FC = () => {
  const { id } = useParams<{ id: string }>();
  const language = usePreferenceStore((state) => state.language);
  const [article, setArticle] = useState<ArticleDetailData | null>(null);
  const [articleContext, setArticleContext] = useState<ArticleContext | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');
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
  const labels = articleDetailLabels(language);
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
        setLoading(false);
        return;
      }

      setLoading(true);
      setError('');
      setContextError('');
      setLikeError('');
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
        setHasLiked(localStorage.getItem(articleLikeStorageKey(res.data.id)) === '1');

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
          setError(getErrorMessage(err, translate(languageRef.current, 'article.loadError')));
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
  }, [id]);

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
      setHasLiked(true);
      localStorage.setItem(articleLikeStorageKey(article.id), '1');
    } catch (err) {
      setLikeError(getErrorMessage(err, labels.likeError));
    } finally {
      setIsLiking(false);
    }
  };

  if (loading && !article) {
    return <ArticleDetailSkeleton />;
  }

  if (!article) {
    return (
      <div className="max-w-md mx-auto mt-20">
        <InlineNotice message={error || t('article.missing')} />
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

      <div className="mx-auto grid w-full max-w-[86rem] gap-8 lg:grid-cols-[16rem_minmax(0,46rem)_16rem] lg:items-start lg:justify-center">
        <div className="hidden lg:block" aria-hidden="true" />

        <article className="min-w-0 w-full">
          {loading && <PagePendingState variant="inline" label={t('common.loadingArticle')} />}
          <InlineNotice message={error} className="mb-6" />
          <header className="mb-16 text-center">
            {coverUrl && (
              <div className="mb-12 w-full h-64 md:h-80 overflow-hidden relative">
                {isCoverHidden ? (
                  <div className="flex h-full items-center justify-center border border-mountain-grey bg-[var(--paper-soft)] text-xs tracking-widest text-ink-light opacity-70">
                    {t('article.coverHidden')}
                  </div>
                ) : (
                  <>
                    <button
                      type="button"
                      className="block h-full w-full cursor-zoom-in bg-transparent p-0"
                      aria-label={labels.viewImage(article.title)}
                      onClick={() => setLightboxImage({ src: coverUrl, alt: article.title })}
                    >
                      <img
                        src={coverUrl}
                        alt={article.title}
                        loading="lazy"
                        decoding="async"
                        onError={() => setCoverError(true)}
                        onLoad={() => setCoverError(false)}
                        className="w-full h-full object-cover grayscale hover:grayscale-0 transition-[filter] duration-slow"
                      />
                    </button>
                    <div className="absolute inset-0 bg-[var(--cover-wash)] pointer-events-none" />
                  </>
                )}
              </div>
            )}

            <h1 className="text-4xl md:text-5xl font-bold text-ink mb-8 leading-tight">
              {article.title}
            </h1>

            <div className="flex flex-col items-center justify-center space-y-4 text-sm text-ink-light tracking-widest opacity-80">
              <div className="flex flex-wrap justify-center gap-x-6 gap-y-2">
                <span>{new Date(article.createdAt).toLocaleDateString(getDateLocale(language), { year: 'numeric', month: 'long', day: 'numeric' })}</span>
                <span>{t('common.views')} {article.viewCount}</span>
                <span>{labels.likes} {likeCount}</span>
              </div>

              <div className="flex flex-wrap items-center justify-center gap-4 text-xs">
                {article.category && (
                  <Link to={`/?categoryId=${article.category.id}`} className="px-2 py-1 border border-mountain-grey text-ochre hover:bg-ochre hover:text-paper transition-colors">
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

          <div className="mt-16 border-t border-mountain-grey border-opacity-50 pt-8 text-center">
            <button
              type="button"
              onClick={handleLike}
              disabled={hasLiked || isLiking}
              className={`group inline-flex items-center gap-3 border px-5 py-2 text-sm tracking-widest transition-colors ${
                hasLiked
                  ? 'border-ochre bg-ochre text-paper'
                  : 'border-mountain-grey text-ink hover:border-ochre hover:text-ochre'
              } disabled:cursor-not-allowed disabled:opacity-80`}
              aria-pressed={hasLiked}
            >
              <span className={`h-2.5 w-2.5 rounded-full ${hasLiked ? 'bg-paper' : 'bg-ochre group-hover:scale-125'} transition-transform`} />
              <span>{hasLiked ? labels.liked : (isLiking ? labels.liking : labels.like)}</span>
              <span>{likeCount}</span>
            </button>
            <InlineNotice message={likeError} className="mt-4" />
          </div>

          <div className="mt-10 text-center">
            <Link to="/" className="inline-block px-6 py-2 border border-ink text-ink hover:bg-ink hover:text-paper transition-colors duration-300 tracking-widest">
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
            className="sticky top-8 hidden lg:block"
          />
        )}
      </div>
    </>
  );
};

const ArticleTOC: React.FC<{
  headings: MarkdownHeading[];
  activeHeadingId: string;
  collapsed: boolean;
  onToggle: () => void;
  language: string;
  className?: string;
}> = ({ headings, activeHeadingId, collapsed, onToggle, language, className = '' }) => {
  if (headings.length === 0) {
    return null;
  }

  const labels = language === 'zh'
    ? { title: '目录', collapse: '收起目录', expand: '展开目录' }
    : { title: 'Contents', collapse: 'Collapse', expand: 'Expand' };

  return (
    <aside className={`border border-mountain-grey bg-[var(--paper-soft)] p-4 ${className}`.trim()} aria-label={labels.title}>
      <div className="mb-3 flex items-center justify-between gap-3">
        <h2 className="text-xs font-bold tracking-[0.2em] text-ink">{labels.title}</h2>
        <button
          type="button"
          onClick={onToggle}
          className="border border-mountain-grey px-2 py-1 text-xs text-ink-light transition-colors hover:border-ochre hover:text-ochre"
          aria-expanded={!collapsed}
        >
          {collapsed ? labels.expand : labels.collapse}
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
                  ? 'border-ochre text-ochre'
                  : 'border-mountain-grey text-ink-light hover:border-ochre hover:text-ink'
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

const ArticleContextBlock: React.FC<{ context: ArticleContext | null; error: string; language: string }> = ({ context, error, language }) => {
  const hasNavigation = Boolean(context?.previous || context?.next);
  const hasRelated = Boolean(context?.related?.length);
  const labels = articleDetailLabels(language);

  if (error) {
    return <InlineNotice message={error} className="mt-10" />;
  }

  if (!context || (!hasNavigation && !hasRelated)) {
    return null;
  }

  return (
    <section className="mt-14 border-t border-mountain-grey border-opacity-50 pt-10">
      {hasNavigation && (
        <div className="grid gap-4 md:grid-cols-2">
          <ArticleNavLink label={labels.previous} article={context.previous} align="left" />
          <ArticleNavLink label={labels.next} article={context.next} align="right" />
        </div>
      )}

      {hasRelated && (
        <div className="mt-12">
          <h2 className="mb-5 text-sm font-bold tracking-widest text-ink">{labels.related}</h2>
          <div className="grid gap-4 md:grid-cols-3">
            {context.related.map((item) => (
              <PreloadLink
                key={item.id}
                to={`/article/${item.id}`}
                preload={routeLoaders.articleDetail}
                className="border border-mountain-grey bg-[var(--paper-soft)] p-4 transition-colors hover:border-ochre"
              >
                <h3 className="line-clamp-2 font-bold leading-relaxed text-ink">{item.title}</h3>
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
      className={`block border border-mountain-grey bg-[var(--paper-soft)] p-4 transition-colors hover:border-ochre ${align === 'right' ? 'md:text-right' : ''}`}
    >
      <span className="text-xs tracking-widest text-ink-light">{label}</span>
      <h2 className="mt-2 line-clamp-2 font-bold leading-relaxed text-ink">{article.title}</h2>
    </PreloadLink>
  );
};

const articleLikeStorageKey = (articleID: number) => `article-like:${articleID}`;

const articleDetailLabels = (language: string) => language === 'zh'
  ? {
      likeError: '点赞失败',
      likes: '点赞',
      liked: '已点赞',
      liking: '提交中',
      like: '点赞',
      previous: '上一篇',
      next: '下一篇',
      related: '相关文章',
      viewImage: (title: string) => `查看大图：${title}`,
    }
  : {
      likeError: 'Failed to like article',
      likes: 'Likes',
      liked: 'Liked',
      liking: 'Liking',
      like: 'Like',
      previous: 'Previous',
      next: 'Next',
      related: 'Related Articles',
      viewImage: (title: string) => `View full-size image: ${title}`,
    };

export default ArticleDetail;
