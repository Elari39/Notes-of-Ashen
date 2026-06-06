import React, { useEffect, useState } from 'react';
import { Link, useSearchParams } from 'react-router-dom';
import Pagination from '../components/Pagination';
import InlineNotice from '../components/InlineNotice';
import { getErrorMessage } from '../utils/error';
import { getDateLocale, translate } from '../i18n';
import { usePreferenceStore } from '../store/preferences';
import { getArticles } from '../api/article';
import { Article } from '../types';

const Home: React.FC = () => {
  const language = usePreferenceStore((state) => state.language);
  const [searchParams, setSearchParams] = useSearchParams();
  const [articles, setArticles] = useState<Article[]>([]);
  const [loading, setLoading] = useState(true);
  const [total, setTotal] = useState(0);
  const [error, setError] = useState('');
  const [keyword, setKeyword] = useState(searchParams.get('q') || '');
  const size = 10;
  const t = (key: Parameters<typeof translate>[1]) => translate(language, key);
  const page = parsePositiveInt(searchParams.get('page'), 1);
  const query = (searchParams.get('q') || '').trim();
  const categoryId = parsePositiveInt(searchParams.get('categoryId'), 0);
  const tagId = parsePositiveInt(searchParams.get('tagId'), 0);
  const hasActiveFilters = Boolean(query || categoryId || tagId);

  useEffect(() => {
    setKeyword(searchParams.get('q') || '');
  }, [searchParams]);

  useEffect(() => {
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
      } catch (err) {
        setError(getErrorMessage(err, translate(language, 'home.loadError')));
      } finally {
        setLoading(false);
      }
    };
    fetchArticles();
  }, [page, query, categoryId, tagId, language]);

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

  const handleSearch = (event: React.FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    updateParams({ q: keyword.trim(), page: undefined });
  };

  const handleClear = () => {
    setKeyword('');
    updateParams({ q: undefined, categoryId: undefined, tagId: undefined, page: undefined });
  };

  return (
    <div className="space-y-20 mt-8 max-w-4xl mx-auto w-full">
      <form onSubmit={handleSearch} className="flex flex-col gap-4 border-b border-mountain-grey border-opacity-50 pb-8 md:flex-row md:items-center">
        <input
          value={keyword}
          onChange={(event) => setKeyword(event.target.value)}
          placeholder={t('home.searchPlaceholder')}
          className="flex-1 bg-transparent border border-mountain-grey px-4 py-3 text-ink outline-none transition-colors placeholder:text-ink-light placeholder:opacity-50 focus:border-ochre"
        />
        <div className="flex gap-3">
          <button type="submit" className="px-5 py-3 border border-ink text-ink hover:bg-ink hover:text-paper transition-colors tracking-widest text-sm">
            {t('home.search')}
          </button>
          {hasActiveFilters && (
            <button type="button" onClick={handleClear} className="px-5 py-3 border border-mountain-grey text-ink-light hover:border-ochre hover:text-ochre transition-colors tracking-widest text-sm">
              {t('home.clearFilters')}
            </button>
          )}
        </div>
      </form>
      <InlineNotice message={error} />
      {loading ? (
        <div className="flex-grow flex items-center justify-center text-ink-light tracking-widest">{t('common.loadingArticles')}</div>
      ) : articles.length === 0 ? (
        <div className="text-center text-ink-light italic">{t('common.emptyArticles')}</div>
      ) : (
        <>
          {articles.map((article) => (
            <article key={article.id} className="group relative flex flex-col md:flex-row gap-8 items-start">
              {article.coverUrl && (
                <div className="w-full md:w-1/3 shrink-0 h-48 overflow-hidden relative">
                  <Link to={`/article/${article.id}`}>
                    <img src={article.coverUrl} alt="cover" className="w-full h-full object-cover grayscale opacity-80 group-hover:grayscale-0 group-hover:opacity-100 transition-all duration-700" />
                  </Link>
                  <div className="absolute inset-0 bg-paper bg-opacity-10 pointer-events-none"></div>
                </div>
              )}
              <div className="flex-1">
                <Link to={`/article/${article.id}`} className="block">
                  <h2 className="text-3xl font-bold text-ink mb-4 group-hover:text-ochre transition-colors duration-500">
                    {article.title}
                  </h2>
                  <p className="text-ink-light leading-relaxed mb-6 whitespace-pre-line line-clamp-3">
                    {article.summary}
                  </p>
                </Link>
                <div className="flex flex-wrap items-center gap-4 text-xs text-ink-light opacity-70 tracking-wider">
                  <span>{new Date(article.createdAt).toLocaleDateString(getDateLocale(language), { year: 'numeric', month: 'long', day: 'numeric' })}</span>
                  <span>{t('common.views')} {article.viewCount}</span>
                  {article.category && (
                    <Link to={`/?categoryId=${article.category.id}`} className="px-2 py-0.5 border border-mountain-grey border-opacity-50 hover:border-ochre hover:text-ochre transition-colors">
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
              <div className="absolute -bottom-10 left-0 w-24 h-px bg-mountain-grey opacity-50 group-hover:w-full group-hover:bg-ochre transition-all duration-700 ease-in-out"></div>
            </article>
          ))}

          <Pagination
            currentPage={page}
            total={total}
            pageSize={size}
            onPageChange={(nextPage) => updateParams({ page: nextPage === 1 ? undefined : nextPage })}
          />
        </>
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
