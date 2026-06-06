import React, { useEffect, useState } from 'react';
import { getCategories } from '../api/category';
import { getTags } from '../api/tag';
import { Category, Tag } from '../types';
import InlineNotice from '../components/InlineNotice';
import { getErrorMessage } from '../utils/error';

const Archive: React.FC = () => {
  const [categories, setCategories] = useState<Category[]>([]);
  const [tags, setTags] = useState<Tag[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');

  useEffect(() => {
    const fetchData = async () => {
      try {
        const [catRes, tagRes] = await Promise.all([
          getCategories({ size: 100 }),
          getTags({ size: 100 })
        ]);
        setCategories(catRes.data.items || []);
        setTags(tagRes.data.items || []);
      } catch (e) {
        setError(getErrorMessage(e, '归档数据加载失败'));
      } finally {
        setLoading(false);
      }
    };
    fetchData();
  }, []);

  if (loading) {
    return <div className="flex-grow flex items-center justify-center text-ink-light tracking-widest">寻梦中...</div>;
  }

  return (
    <div className="space-y-16 mt-8 max-w-2xl mx-auto w-full">
      <InlineNotice message={error} />
      <section>
        <h2 className="text-3xl font-bold text-ink mb-8 tracking-widest text-center">分类</h2>
        <div className="flex flex-wrap gap-4 justify-center">
          {categories.length === 0 && <span className="text-ink-light opacity-50 italic">暂无分类</span>}
          {categories.map(c => (
            <span key={c.id} className="px-4 py-2 border border-mountain-grey text-ink-light hover:border-ochre hover:text-ochre transition-colors cursor-pointer tracking-wider">
              {c.name}
            </span>
          ))}
        </div>
      </section>

      <section>
        <h2 className="text-3xl font-bold text-ink mb-8 tracking-widest text-center">标签</h2>
        <div className="flex flex-wrap gap-4 justify-center">
          {tags.length === 0 && <span className="text-ink-light opacity-50 italic">暂无标签</span>}
          {tags.map(t => (
            <span key={t.id} className="text-ink hover:text-ochre transition-colors cursor-pointer relative before:content-['#'] before:mr-1 before:opacity-30 tracking-wider">
              {t.name}
            </span>
          ))}
        </div>
      </section>
    </div>
  );
};

export default Archive;
