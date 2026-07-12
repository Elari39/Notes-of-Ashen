import React, { useEffect, useState } from 'react';
import { Link } from 'react-router-dom';
import { getCategories } from '../api/category';
import { getTags } from '../api/tag';
import { Category, Tag } from '../types';
import InlineNotice from '../components/InlineNotice';
import PagePendingState from '../components/RoutePending';
import Skeleton from '../components/Skeleton';
import EmptyState from '../components/ui/EmptyState';
import Button from '../components/ui/Button';
import { getErrorMessage } from '../utils/error';
import { translate } from '../i18n';
import { usePreferenceStore } from '../store/preferences';

const Archive: React.FC = () => {
  const language = usePreferenceStore((state) => state.language);
  const [categories, setCategories] = useState<Category[]>([]);
  const [tags, setTags] = useState<Tag[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');
  const [retryVersion, setRetryVersion] = useState(0);
  const t = (key: Parameters<typeof translate>[1]) => translate(language, key);

  useEffect(() => {
    let active = true;
    const fetchData = async () => {
      setLoading(true);
      setError('');
      try {
        const [catRes, tagRes] = await Promise.all([
          getCategories({ size: 100 }),
          getTags({ size: 100 }),
        ]);
        if (!active) {
          return;
        }
        setCategories(catRes.data.items || []);
        setTags(tagRes.data.items || []);
      } catch (e) {
        if (active) {
          setError(getErrorMessage(e, translate(language, 'archive.loadError')));
        }
      } finally {
        if (active) {
          setLoading(false);
        }
      }
    };
    fetchData();
    return () => {
      active = false;
    };
  }, [language, retryVersion]);

  const hasArchiveData = categories.length > 0 || tags.length > 0;
  const handleRetry = () => {
    setRetryVersion((version) => version + 1);
  };

  return (
    <div className="editorial-container w-full space-y-12">
      <header className="max-w-3xl py-6 md:py-10">
        <p className="editorial-kicker">{t('archive.kicker')}</p>
        <h1 className="mt-5 editorial-page-title">{t('nav.archive')}</h1>
      </header>
      <InlineNotice
        message={error}
        action={(
          <Button type="button" variant="ghost" size="sm" onClick={handleRetry}>
            {t('common.retry')}
          </Button>
        )}
      />
      {loading && !hasArchiveData && (
        <div className="grid gap-6 md:grid-cols-2">
          {[0, 1].map((section) => (
            <section key={section} className="editorial-card">
              <Skeleton className="mx-auto mb-8 h-7 w-32" />
              <div className="flex flex-wrap gap-4 justify-center">
                {Array.from({ length: 8 }).map((_, i) => (
                  <Skeleton key={i} className="h-9 w-20" />
                ))}
              </div>
            </section>
          ))}
        </div>
      )}
      {loading && hasArchiveData && (
        <PagePendingState variant="inline" label={t('common.loadingArchive')} />
      )}
      {!loading && (!error || hasArchiveData) && (
        <>
          <div className="grid gap-6 md:grid-cols-2">
          <section className="editorial-card">
            <h2 className="mb-8 font-display text-4xl text-ink">{t('archive.titleCategories')}</h2>
            {categories.length === 0 ? (
              <EmptyState illustration="leaf" title={t('common.noCategory')} />
            ) : (
              <div className="flex flex-wrap gap-3">
                {categories.map(c => (
                  <Link key={c.id} to={`/?categoryId=${c.id}`} className="inline-flex min-h-11 items-center rounded-md border border-hairline bg-paper px-4 py-2 text-sm font-medium text-ink transition-colors duration-base hover:border-ink focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-ochre">
                    {c.name}
                  </Link>
                ))}
              </div>
            )}
          </section>

          <section className="editorial-dark-card">
            <h2 className="mb-8 font-display text-4xl text-on-dark">{t('archive.titleTags')}</h2>
            {tags.length === 0 ? (
              <EmptyState illustration="leaf" title={t('common.noTag')} />
            ) : (
              <div className="flex flex-wrap gap-3">
                {tags.map(tg => (
                  <Link key={tg.id} to={`/?tagId=${tg.id}`} className="relative inline-flex min-h-11 items-center rounded-full bg-surface-dark-soft px-4 py-2 text-sm text-on-dark transition-colors duration-base before:mr-1 before:opacity-50 before:content-['#'] hover:text-ochre focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-ochre">
                    {tg.name}
                  </Link>
                ))}
              </div>
            )}
          </section>
          </div>
        </>
      )}
    </div>
  );
};

export default Archive;
