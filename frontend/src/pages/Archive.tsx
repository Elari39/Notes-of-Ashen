import React, { useEffect, useState } from 'react';
import { Link } from 'react-router-dom';
import { getCategories } from '../api/category';
import { getTags } from '../api/tag';
import { Category, Tag } from '../types';
import InlineNotice from '../components/InlineNotice';
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
    const fetchData = async () => {
      try {
        const [catRes, tagRes] = await Promise.all([
          getCategories({ size: 100 }),
          getTags({ size: 100 }),
        ]);
        setCategories(catRes.data.items || []);
        setTags(tagRes.data.items || []);
      } catch (e) {
        setError(getErrorMessage(e, translate(language, 'archive.loadError')));
      } finally {
        setLoading(false);
      }
    };
    fetchData();
  }, [language]);

  if (loading) {
    return <div className="flex-grow flex items-center justify-center text-ink-light tracking-widest">{t('common.loadingArchive')}</div>;
  }

  return (
    <div className="space-y-16 mt-8 max-w-2xl mx-auto w-full">
      <InlineNotice message={error} />
      <section>
        <h2 className="text-3xl font-bold text-ink mb-8 tracking-widest text-center">{t('archive.titleCategories')}</h2>
        <div className="flex flex-wrap gap-4 justify-center">
          {categories.length === 0 && <span className="text-ink-light opacity-50 italic">{t('common.noCategory')}</span>}
          {categories.map(c => (
            <Link key={c.id} to={`/?categoryId=${c.id}`} className="px-4 py-2 border border-mountain-grey text-ink-light hover:border-ochre hover:text-ochre transition-colors tracking-wider">
              {c.name}
            </Link>
          ))}
        </div>
      </section>

      <section>
        <h2 className="text-3xl font-bold text-ink mb-8 tracking-widest text-center">{t('archive.titleTags')}</h2>
        <div className="flex flex-wrap gap-4 justify-center">
          {tags.length === 0 && <span className="text-ink-light opacity-50 italic">{t('common.noTag')}</span>}
          {tags.map(tg => (
            <Link key={tg.id} to={`/?tagId=${tg.id}`} className="text-ink hover:text-ochre transition-colors relative before:content-['#'] before:mr-1 before:opacity-30 tracking-wider">
              {tg.name}
            </Link>
          ))}
        </div>
      </section>
    </div>
  );
};

export default Archive;
