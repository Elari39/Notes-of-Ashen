import React, { useCallback, useEffect, useState } from 'react';
import { useParams, Link } from 'react-router-dom';
import ImageLightbox, { LightboxImage } from '../components/ImageLightbox';
import InlineNotice from '../components/InlineNotice';
import MarkdownRenderer from '../components/MarkdownRenderer';
import { getErrorMessage } from '../utils/error';
import { getArticleById, getArticleContext } from '../api/article';
import { useSEO } from '../utils/seo';
import { Article, ArticleContext } from '../types';
import { getDateLocale, translate } from '../i18n';
import { usePreferenceStore } from '../store/preferences';
import { normalizeCoverUrl } from '../utils/cover';

type ArticleDetailData = Article & { content: string };

const ArticleDetail: React.FC = () => {
  const { id } = useParams<{ id: string }>();
  const language = usePreferenceStore((state) => state.language);
  const [article, setArticle] = useState<ArticleDetailData | null>(null);
  const [articleContext, setArticleContext] = useState<ArticleContext | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');
  const [contextError, setContextError] = useState('');
  const [coverError, setCoverError] = useState(false);
  const [lightboxImage, setLightboxImage] = useState<LightboxImage | null>(null);
  const closeLightbox = useCallback(() => setLightboxImage(null), []);
  const t = (key: Parameters<typeof translate>[1]) => translate(language, key);

  useSEO(article?.seoTitle || article?.title, article?.seoDescription || article?.summary, article?.seoKeywords);

  useEffect(() => {
    const fetchArticle = async () => {
      if (!id) {
        setLoading(false);
        return;
      }

      setLoading(true);
      setError('');
      setContextError('');
      setArticle(null);
      setArticleContext(null);
      setCoverError(false);
      setLightboxImage(null);
      try {
        const res = await getArticleById(id);
        setArticle({ ...res.data, content: res.data.content || '' });
        try {
          const contextRes = await getArticleContext(id);
          setArticleContext(contextRes.data);
        } catch (contextErr) {
          setContextError(getErrorMessage(contextErr, translate(language, 'article.loadError')));
        }
      } catch (err) {
        setError(getErrorMessage(err, translate(language, 'article.loadError')));
      } finally {
        setLoading(false);
      }
    };
    fetchArticle();
  }, [id, language]);

  if (loading) {
    return <div className="flex-grow flex items-center justify-center text-ink-light tracking-widest">{t('common.loadingArticle')}</div>;
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
    <article className="max-w-2xl mx-auto w-full">
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
                  aria-label={`查看大图：${article.title}`}
                  onClick={() => setLightboxImage({ src: coverUrl, alt: article.title })}
                >
                  <img
                    src={coverUrl}
                    alt={article.title}
                    onError={() => setCoverError(true)}
                    onLoad={() => setCoverError(false)}
                    className="w-full h-full object-cover grayscale hover:grayscale-0 transition-all duration-700"
                  />
                </button>
                <div className="absolute inset-0 bg-[var(--cover-wash)] pointer-events-none"></div>
              </>
            )}
          </div>
        )}
        <h1 className="text-4xl md:text-5xl font-bold text-ink mb-8 leading-tight">
          {article.title}
        </h1>
        <div className="flex flex-col items-center justify-center space-y-4 text-sm text-ink-light tracking-widest opacity-80">
          <div className="flex space-x-6">
            <span>{new Date(article.createdAt).toLocaleDateString(getDateLocale(language), { year: 'numeric', month: 'long', day: 'numeric' })}</span>
            <span>{t('common.views')} {article.viewCount}</span>
          </div>
          <div className="flex items-center space-x-4 text-xs">
            {article.category && (
              <Link to={`/?categoryId=${article.category.id}`} className="px-2 py-1 border border-mountain-grey text-ochre hover:bg-ochre hover:text-paper transition-colors">
                {article.category.name}
              </Link>
            )}
            {article.tags && article.tags.length > 0 && (
              <div className="flex space-x-3">
                {article.tags.map(tg => (
                  <Link key={tg.id} to={`/?tagId=${tg.id}`} className="relative hover:text-ochre transition-colors before:content-['#'] before:mr-1 before:opacity-30">
                    {tg.name}
                  </Link>
                ))}
              </div>
            )}
          </div>
        </div>
      </header>

      <MarkdownRenderer content={article.content} className="prose-lg mx-auto" />

      <div className="mt-20 pt-8 border-t border-mountain-grey border-opacity-50 text-center">
        <Link to="/" className="inline-block px-6 py-2 border border-ink text-ink hover:bg-ink hover:text-paper transition-colors duration-300 tracking-widest">
          {t('common.backHome')}
        </Link>
      </div>

      <ArticleContextBlock context={articleContext} error={contextError} />
      <ImageLightbox image={lightboxImage} onClose={closeLightbox} />
    </article>
  );
};

const ArticleContextBlock: React.FC<{ context: ArticleContext | null; error: string }> = ({ context, error }) => {
  const hasNavigation = Boolean(context?.previous || context?.next);
  const hasRelated = Boolean(context?.related?.length);

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
          <ArticleNavLink label="上一篇" article={context.previous} align="left" />
          <ArticleNavLink label="下一篇" article={context.next} align="right" />
        </div>
      )}

      {hasRelated && (
        <div className="mt-12">
          <h2 className="mb-5 text-sm font-bold tracking-widest text-ink">相关文章</h2>
          <div className="grid gap-4 md:grid-cols-3">
            {context.related.map((item) => (
              <Link
                key={item.id}
                to={`/article/${item.id}`}
                className="border border-mountain-grey bg-[var(--paper-soft)] p-4 transition-colors hover:border-ochre"
              >
                <h3 className="line-clamp-2 font-bold leading-relaxed text-ink">{item.title}</h3>
                <p className="mt-3 line-clamp-3 text-sm leading-relaxed text-ink-light">{item.summary}</p>
              </Link>
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
    <Link
      to={`/article/${article.id}`}
      className={`block border border-mountain-grey bg-[var(--paper-soft)] p-4 transition-colors hover:border-ochre ${align === 'right' ? 'md:text-right' : ''}`}
    >
      <span className="text-xs tracking-widest text-ink-light">{label}</span>
      <h2 className="mt-2 line-clamp-2 font-bold leading-relaxed text-ink">{article.title}</h2>
    </Link>
  );
};

export default ArticleDetail;
