import { Link } from 'react-router-dom';
import type { Article, HomeArticleLayout } from '../types';
import { PreloadLink } from './PreloadLink';
import Tag from './ui/Tag';
import { normalizeCoverUrl } from '../utils/cover';
import { formatText, getDateLocale, translate } from '../i18n';
import { usePreferenceStore } from '../store/preferences';
import { routeLoaders } from '../routes/lazyRoutes';

export default function HomeArticleCard({ article, layout, position, reverseCover, coverFailed, onCoverError }: {
  article: Article; layout: HomeArticleLayout; position: number; reverseCover: boolean; coverFailed: boolean; onCoverError: (id: number) => void;
}) {
  const language = usePreferenceStore((state) => state.language);
  const t = (key: Parameters<typeof translate>[1]) => translate(language, key);
  const coverUrl = normalizeCoverUrl(article.coverUrl);
  const shouldShowCover = Boolean(coverUrl && !coverFailed);
  const shouldReverse = layout === 'alternating' && reverseCover;
  const titleClass = 'text-2xl sm:text-3xl';
  return (
    <article
      className={`home-article-card group relative overflow-hidden rounded-lg border border-hairline bg-surface-card shadow-xs transition-[transform,box-shadow] duration-base hover:-translate-y-0.5 hover:shadow-sm motion-reduce:transform-none ${layout === 'alternating' ? `flex flex-col items-stretch md:min-h-72 md:flex-row ${shouldReverse ? 'md:flex-row-reverse' : ''}` : 'flex flex-col'}`}
    >
      {shouldShowCover && (
        <div className={`relative aspect-[16/9] w-full shrink-0 overflow-hidden ${layout === 'alternating' ? 'md:h-auto md:w-[42%] md:aspect-auto' : ''}`}>
          <img
            src={coverUrl}
            alt=""
            loading="lazy"
            decoding="async"
            onError={() => onCoverError(article.id)}
            className="h-full w-full object-cover opacity-90 transition-[opacity,transform] duration-slow group-hover:scale-[1.015] group-hover:opacity-100 motion-reduce:transform-none motion-reduce:transition-none"
          />
          <div className="pointer-events-none absolute inset-0 bg-[var(--cover-wash-subtle)]"></div>
        </div>
      )}

      <div className="home-article-body flex min-w-0 flex-1 flex-col justify-between p-5 sm:p-6 md:p-8">
        <span className="home-article-index" aria-hidden="true">{String(position).padStart(2, '0')}</span>
        <PreloadLink
          to={`/article/${article.id}`}
          preload={routeLoaders.articleDetail}
          className="block rounded-sm focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-4 focus-visible:outline-ochre"
        >
          <h3 className={`mb-4 font-display font-normal leading-[1.08] tracking-[-0.02em] text-ink transition-colors duration-base group-hover:text-ochre ${titleClass}`}>
            {article.title}
          </h3>
          {article.summary && (
            <p className="mb-7 line-clamp-3 whitespace-pre-line text-sm leading-7 text-body">
              {article.summary}
            </p>
          )}
        </PreloadLink>

        <div className="space-y-4 text-xs text-muted">
          <div className="flex flex-wrap items-center gap-x-4 gap-y-2">
            {article.isPinned && <Tag tone="ochre" size="sm">{t('common.pinned')}</Tag>}
            <span>{new Date(article.publishedAt || article.createdAt).toLocaleDateString(getDateLocale(language), { year: 'numeric', month: 'long', day: 'numeric' })}</span>
            <span>{t('common.views')} {article.viewCount}</span>
            <span>{formatText(t('reading.minutes'), { count: article.readingTimeMinutes })}</span>
          </div>

          {(article.category || (article.tags && article.tags.length > 0)) && (
            <div className="flex flex-wrap items-center gap-2">
              {article.category && (
                <Link
                  to={`/?categoryId=${article.category.id}`}
                  className="theme-category inline-flex min-h-11 items-center rounded-full bg-paper px-4 py-2 font-medium text-ink transition-colors hover:text-ochre focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-ochre"
                >
                  {article.category.name}
                </Link>
              )}
              {article.tags?.map((tag) => (
                <Link
                  key={tag.id}
                  to={`/?tagId=${tag.id}`}
                  className="inline-flex min-h-11 items-center rounded-full px-3 py-2 transition-colors before:mr-1 before:content-['#'] before:opacity-40 hover:text-ochre focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-ochre"
                >
                  {tag.name}
                </Link>
              ))}
            </div>
          )}
        </div>
      </div>
    </article>
  );
}
