import React, { useEffect, useState } from 'react';
import { Link, useParams } from 'react-router-dom';
import { getArticlePreview } from '../../api/article';
import InlineNotice from '../../components/InlineNotice';
import MarkdownRenderer from '../../components/MarkdownRenderer';
import { getErrorMessage } from '../../utils/error';
import { useSEO } from '../../utils/seo';
import type { Article } from '../../types';

const ArticlePreview: React.FC = () => {
  const { id } = useParams();
  const [article, setArticle] = useState<Article | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');

  useSEO(article?.seoTitle || article?.title, article?.seoDescription || article?.summary, article?.seoKeywords);

  useEffect(() => {
    const fetchPreview = async () => {
      if (!id) {
        return;
      }
      setLoading(true);
      setError('');
      try {
        const res = await getArticlePreview(id);
        setArticle(res.data);
      } catch (e) {
        setError(getErrorMessage(e, '预览加载失败'));
      } finally {
        setLoading(false);
      }
    };
    fetchPreview();
  }, [id]);

  if (loading) {
    return <div className="py-20 text-center tracking-widest text-ink-light">预览加载中...</div>;
  }

  if (!article) {
    return <InlineNotice message={error || '文章不存在'} />;
  }

  return (
    <article className="mx-auto max-w-3xl">
      <div className="mb-8 flex flex-wrap items-center justify-between gap-3 border border-ochre px-4 py-3 text-sm text-ochre">
        <span>草稿预览，不会增加阅读量</span>
        <Link to={`/admin/editor/${article.id}`} className="hover:text-ink">返回编辑</Link>
      </div>
      <h1 className="mb-4 text-4xl font-bold text-ink">{article.title}</h1>
      <p className="mb-10 text-ink-light">{article.summary}</p>
      <MarkdownRenderer content={article.content || ''} />
    </article>
  );
};

export default ArticlePreview;
