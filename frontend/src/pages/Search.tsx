import React, { useEffect, useState } from 'react';
import { Link, useSearchParams } from 'react-router-dom';
import InlineNotice from '../components/InlineNotice';
import Pagination from '../components/Pagination';
import SearchHighlight from '../components/SearchHighlight';
import { getArticles } from '../api/article';
import { getErrorMessage } from '../utils/error';
import { normalizeCoverUrl } from '../utils/cover';
import { Article } from '../types';
import { formatText, getDateLocale, translate } from '../i18n';
import { usePreferenceStore } from '../store/preferences';

const Search: React.FC = () => {
  const language = usePreferenceStore((state) => state.language);
  const [searchParams, setSearchParams] = useSearchParams();
  const [keyword, setKeyword] = useState('');
  const [articles, setArticles] = useState<Article[]>([]);
  const [loading, setLoading] = useState(false);
  const [total, setTotal] = useState(0);
  const [error, setError] = useState('');
  const [coverErrors, setCoverErrors] = useState<Record<number, boolean>>({});
  const size = 10;
  const t = (key: Parameters<typeof translate>[1]) => translate(language, key);
  const pinnedLabel = language === 'zh' ? '\u7f6e\u9876' : 'Pinned';
  const query = (searchParams.get('q') || '').trim();
  const page = parsePositiveInt(searchParams.get('page'), 1);
  const categoryId = parsePositiveInt(searchParams.get('categoryId'), 0);
  const tagId = parsePositiveInt(searchParams.get('tagId'), 0);
  const hasSearchScope = Boolean(query || categoryId || tagId);

  useEffect(() => {
    setKeyword(query);
  }, [query]);

  useEffect(() => {
    if (!hasSearchScope) {
      setArticles([]);
      setTotal(0);
      setError('');
      setLoading(false);
      setCoverErrors({});
      return;
    }

    const fetchArticles = async () => {
      setLoading(true);
      setError('');
      try {
        const res = await getArticles({
          status: 'published',
          page,
          size,
          ...(query ? { q: query } : {}),
          ...(categoryId ? { categoryId } : {}),
          ...(tagId ? { tagId } : {}),
        });
        setArticles(res.data.items || []);
        setTotal(res.data.total || 0);
        setCoverErrors({});
      } catch (err) {
        setError(getErrorMessage(err, translate(language, 'search.loadError')));
      } finally {
        setLoading(false);
      }
    };

    fetchArticles();
  }, [hasSearchScope, page, query, categoryId, tagId, language]);

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

  const handleSubmit = (event: React.FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    const nextKeyword = keyword.trim();

    if (!nextKeyword) {
      setSearchParams({});
      return;
    }

    setSearchParams({ q: nextKeyword });
  };

  const handleClear = () => {
    setKeyword('');
    setSearchParams({});
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
    <div className="mx-auto mt-4 w-full max-w-5xl">
      <section className="relative overflow-hidden border-y border-mountain-grey border-opacity-70 py-12 md:py-16">
        <div className="absolute left-0 top-8 h-24 w-px bg-ochre opacity-60"></div>
        <div className="absolute bottom-8 right-0 h-24 w-px bg-ochre opacity-40"></div>

        <div className="mx-auto max-w-3xl text-center">
          <p className="text-xs tracking-[0.32em] text-ochre">{t('search.kicker')}</p>
          <h1 className="mt-5 text-4xl font-bold tracking-widest text-ink md:text-5xl">
            {t('search.title')}
          </h1>
          <p className="mx-auto mt-5 max-w-2xl text-sm leading-loose tracking-wide text-ink-light">
            {t('search.subtitle')}
          </p>

          <form onSubmit={handleSubmit} className="mx-auto mt-10 flex max-w-2xl flex-col gap-3 border border-mountain-grey bg-[var(--paper-muted)] p-3 md:flex-row md:items-center">
            <input
              value={keyword}
              onChange={(event) => setKeyword(event.target.value)}
              aria-label={t('search.inputLabel')}
              placeholder={t('search.placeholder')}
              className="min-w-0 flex-1 bg-transparent px-3 py-3 text-lg text-ink outline-none placeholder:text-ink-light placeholder:opacity-50"
            />
            <div className="flex gap-3">
              {hasSearchScope && (
                <button
                  type="button"
                  onClick={handleClear}
                  className="border border-mountain-grey px-4 py-3 text-sm tracking-widest text-ink-light transition-colors hover:border-ochre hover:text-ochre"
                >
                  {t('search.clear')}
                </button>
              )}
              <button
                type="submit"
                className="border border-ink bg-ink px-5 py-3 text-sm tracking-widest text-paper transition-colors hover:border-ochre hover:bg-ochre"
              >
                {t('search.submit')}
              </button>
            </div>
          </form>
        </div>
      </section>

      <section className="mt-12">
        {!hasSearchScope ? (
          <div className="mx-auto max-w-xl text-center text-ink-light">
            <p className="text-lg italic">{t('search.idleTitle')}</p>
            <p className="mt-3 text-sm leading-loose opacity-80">{t('search.idleText')}</p>
          </div>
        ) : (
          <div className="space-y-12">
            <div className="flex flex-col gap-3 border-b border-mountain-grey border-opacity-50 pb-5 text-sm text-ink-light md:flex-row md:items-end md:justify-between">
              <div>
                <p className="tracking-widest text-ochre">{t('search.resultsKicker')}</p>
                <p className="mt-2 tracking-wide">
                  {query
                    ? formatText(t('search.resultsFor'), { keyword: query, total })
                    : formatText(t('search.resultsFiltered'), { total })}
                </p>
              </div>
              <button
                type="button"
                onClick={handleClear}
                className="self-start border border-mountain-grey px-4 py-2 tracking-widest transition-colors hover:border-ochre hover:text-ochre md:self-auto"
              >
                {t('search.clear')}
              </button>
            </div>

            <InlineNotice message={error} />

            {loading ? (
              <div className="flex min-h-48 items-center justify-center text-ink-light tracking-widest">{t('search.loading')}</div>
            ) : articles.length === 0 ? (
              <div className="mx-auto max-w-xl py-10 text-center text-ink-light">
                <p className="text-lg italic">{t('search.emptyTitle')}</p>
                <p className="mt-3 text-sm leading-loose opacity-80">{t('search.emptyText')}</p>
              </div>
            ) : (
              <>
                <div className="space-y-12">
                  {articles.map((article) => {
                    const coverUrl = normalizeCoverUrl(article.coverUrl);
                    const isCoverHidden = Boolean(coverUrl && coverErrors[article.id]);

                    return (
                      <article key={article.id} className="group grid gap-6 border-b border-mountain-grey border-opacity-40 pb-12 md:grid-cols-[12rem_1fr]">
                        {coverUrl ? (
                          <Link to={`/article/${article.id}`} className="block h-44 overflow-hidden">
                            {isCoverHidden ? (
                              <div className="flex h-full items-center justify-center border border-mountain-grey bg-[var(--paper-soft)] text-xs tracking-widest text-ink-light opacity-70">
                                {t('article.coverHidden')}
                              </div>
                            ) : (
                              <img
                                src={coverUrl}
                                alt={article.title}
                                onError={() => handleCoverError(article.id)}
                                onLoad={() => handleCoverLoad(article.id)}
                                className="h-full w-full object-cover grayscale opacity-80 transition-all duration-700 group-hover:grayscale-0 group-hover:opacity-100"
                              />
                            )}
                          </Link>
                        ) : (
                          <div className="hidden h-44 border border-mountain-grey bg-[var(--paper-soft)] md:block"></div>
                        )}

                        <div>
                          <Link to={`/article/${article.id}`} className="block">
                            <h2 className="text-2xl font-bold leading-tight text-ink transition-colors duration-500 group-hover:text-ochre md:text-3xl">
                              <SearchHighlight value={article.searchHighlights?.title} fallback={article.title} />
                            </h2>
                            <p className="mt-4 line-clamp-3 whitespace-pre-line text-ink-light leading-relaxed">
                              <SearchHighlight
                                value={article.searchHighlights?.summary || article.searchHighlights?.content}
                                fallback={article.summary}
                              />
                            </p>
                          </Link>

                          <div className="mt-5 flex flex-wrap items-center gap-4 text-xs tracking-wider text-ink-light opacity-75">
                            {article.isPinned && (
                              <span className="border border-ochre px-2 py-0.5 text-ochre opacity-90">{pinnedLabel}</span>
                            )}
                            <span>{new Date(article.createdAt).toLocaleDateString(getDateLocale(language), { year: 'numeric', month: 'long', day: 'numeric' })}</span>
                            <span>{t('common.views')} {article.viewCount}</span>
                            {article.category && (
                              <Link to={`/?categoryId=${article.category.id}`} className="border border-mountain-grey border-opacity-60 px-2 py-0.5 transition-colors hover:border-ochre hover:text-ochre">
                                {article.category.name}
                              </Link>
                            )}
                            {article.tags && article.tags.length > 0 && (
                              <div className="flex flex-wrap gap-3">
                                {article.tags.map((tag) => (
                                  <Link key={tag.id} to={`/?tagId=${tag.id}`} className="relative transition-colors before:mr-1 before:opacity-30 before:content-['#'] hover:text-ochre">
                                    {tag.name}
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
            )}
          </div>
        )}
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

export default Search;
