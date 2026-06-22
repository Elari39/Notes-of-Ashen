import React, { useEffect, useRef, useState } from 'react';
import { Link, useParams } from 'react-router-dom';
import { getArticlePreview } from '../../api/article';
import InlineNotice from '../../components/InlineNotice';
import MarkdownRenderer from '../../components/MarkdownRenderer';
import PagePendingState from '../../components/RoutePending';
import { getErrorMessage } from '../../utils/error';
import { usePreferenceStore } from '../../store/preferences';
import { useSEO } from '../../utils/seo';
import type { Article } from '../../types';

const ArticlePreview: React.FC = () => {
  const { id } = useParams();
  const language = usePreferenceStore((state) => state.language);
  const labels = articlePreviewLabels(language);
  const languageRef = useRef(language);
  languageRef.current = language;
  const [article, setArticle] = useState<Article | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');

  useSEO(article?.seoTitle || article?.title, article?.seoDescription || article?.summary, article?.seoKeywords);

  useEffect(() => {
    let active = true;
    const fetchPreview = async () => {
      if (!id) {
        return;
      }
      setLoading(true);
      setError('');
      try {
        const res = await getArticlePreview(id);
        if (!active) {
          return;
        }
        setArticle(res.data);
      } catch (e) {
        if (!active) {
          return;
        }
        setError(getErrorMessage(e, articlePreviewLabels(languageRef.current).loadError));
      } finally {
        if (active) {
          setLoading(false);
        }
      }
    };
    fetchPreview();
    return () => {
      active = false;
    };
  }, [id]);

  if (loading) {
    return <PagePendingState variant="admin" label={labels.loading} />;
  }

  if (!article) {
    return <InlineNotice message={error || labels.missing} />;
  }

  return (
    <article className="mx-auto max-w-3xl">
      <div className="mb-8 flex flex-wrap items-center justify-between gap-3 border border-ochre px-4 py-3 text-sm text-ochre">
        <span>{labels.notice}</span>
        <Link to={`/admin/editor/${article.id}`} className="hover:text-ink">{labels.backToEditor}</Link>
      </div>
      <h1 className="mb-4 text-4xl font-bold text-ink">{article.title}</h1>
      <p className="mb-10 text-ink-light">{article.summary}</p>
      <MarkdownRenderer content={article.content || ''} />
    </article>
  );
};

const articlePreviewLabels = (language: string) => language === 'zh'
  ? {
      loading: '预览加载中...',
      loadError: '预览加载失败',
      missing: '文章不存在',
      notice: '草稿预览，不会增加阅读量',
      backToEditor: '返回编辑',
    }
  : {
      loading: 'Loading preview...',
      loadError: 'Failed to load preview',
      missing: 'Article not found',
      notice: 'Draft preview. Views will not be counted.',
      backToEditor: 'Back to Editor',
    };

export default ArticlePreview;
