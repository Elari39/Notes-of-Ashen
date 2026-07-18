import React, { useEffect, useRef, useState } from 'react';
import { Link, useParams } from 'react-router-dom';
import { getArticlePreview } from '../../api/article';
import InlineNotice from '../../components/InlineNotice';
import DeferredMarkdownRenderer from '../../components/DeferredMarkdownRenderer';
import PagePendingState from '../../components/RoutePending';
import { getErrorMessage } from '../../utils/error';
import { usePreferenceStore } from '../../store/preferences';
import { translate } from '../../i18n';
import { useSEO } from '../../utils/seo';
import type { Article } from '../../types';

const ArticlePreview: React.FC = () => {
  const { id } = useParams();
  const language = usePreferenceStore((state) => state.language);
  const t = (key: Parameters<typeof translate>[1]) => translate(language, key);
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
        setError(getErrorMessage(e, translate(languageRef.current, 'articlePreview.loadError')));
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
    return <PagePendingState variant="admin" label={t('articlePreview.loading')} />;
  }

  if (!article) {
    return <InlineNotice message={error || t('articlePreview.missing')} />;
  }

  return (
    <article className="mx-auto max-w-3xl">
      <div className="mb-8 flex flex-wrap items-center justify-between gap-3 border border-ochre px-4 py-3 text-sm text-ochre">
        <span>{t('articlePreview.notice')}</span>
        <Link to={`/admin/editor/${article.id}`} className="hover:text-ink">{t('articlePreview.backToEditor')}</Link>
      </div>
      <h1 className="mb-4 text-4xl font-bold text-ink">{article.title}</h1>
      <p className="mb-10 text-ink-light">{article.summary}</p>
      <DeferredMarkdownRenderer content={article.content || ''} />
    </article>
  );
};

export default ArticlePreview;
