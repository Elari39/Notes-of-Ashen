import React, { useEffect, useState } from 'react';
import { Link } from 'react-router-dom';
import { getCategories } from '../api/category';
import { getTags } from '../api/tag';
import { Category, Tag } from '../types';
import InlineNotice from '../components/InlineNotice';
import PagePendingState from '../components/RoutePending';
import Skeleton from '../components/Skeleton';
import EmptyState from '../components/ui/EmptyState';
import { getErrorMessage } from '../utils/error';
import { translate } from '../i18n';
import { usePreferenceStore } from '../store/preferences';

const Archive: React.FC = () => {
  const language = usePreferenceStore((state) => state.language);
  const [categories, setCategories] = useState<Category[]>([]);
  const [tags, setTags] = useState<Tag[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');
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
  }, [language]);

  const hasArchiveData = categories.length > 0 || tags.length > 0;

  return (
    <div className="space-y-16 mt-8 max-w-2xl mx-auto w-full">
      <InlineNotice message={error} />
      {loading && !hasArchiveData && (
        <div className="space-y-16">
          {[0, 1].map((section) => (
            <section key={section}>
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
      {!loading && (
        <>
          <section>
            <h2 className="text-3xl font-bold text-ink mb-8 tracking-widest text-center">{t('archive.titleCategories')}</h2>
            {categories.length === 0 ? (
              <EmptyState illustration="leaf" title={t('common.noCategory')} />
            ) : (
              <div className="flex flex-wrap gap-4 justify-center">
                {categories.map(c => (
                  <Link key={c.id} to={`/?categoryId=${c.id}`} className="px-4 py-2 border border-mountain-grey text-ink-light hover:border-ochre hover:text-ochre transition-colors duration-base tracking-wider focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-ochre">
                    {c.name}
                  </Link>
                ))}
              </div>
            )}
          </section>

          <section>
            <h2 className="text-3xl font-bold text-ink mb-8 tracking-widest text-center">{t('archive.titleTags')}</h2>
            {tags.length === 0 ? (
              <EmptyState illustration="leaf" title={t('common.noTag')} />
            ) : (
              <div className="flex flex-wrap gap-4 justify-center">
                {tags.map(tg => (
                  <Link key={tg.id} to={`/?tagId=${tg.id}`} className="text-ink hover:text-ochre transition-colors duration-base relative before:content-['#'] before:mr-1 before:opacity-30 tracking-wider focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-ochre">
                    {tg.name}
                  </Link>
                ))}
              </div>
            )}
          </section>
        </>
      )}
    </div>
  );
};

export default Archive;
