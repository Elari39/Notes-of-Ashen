import React, { useEffect, useState } from 'react';
import { useParams, useNavigate } from 'react-router-dom';
import ReactMarkdown from 'react-markdown';
import { Components } from 'react-markdown';
import { Prism as SyntaxHighlighter } from 'react-syntax-highlighter';
import { vs } from 'react-syntax-highlighter/dist/esm/styles/prism';
import { getArticleById, createArticle, updateArticle } from '../../api/article';
import { getCategories } from '../../api/category';
import { getTags } from '../../api/tag';
import { Category, Tag } from '../../types';
import InlineNotice from '../../components/InlineNotice';
import { getErrorMessage } from '../../utils/error';
import { getArticleStatusLabel, translate } from '../../i18n';
import { usePreferenceStore } from '../../store/preferences';

const markdownComponents: Components = {
  code({ className, children, ...props }) {
    const match = /language-(\w+)/.exec(className || '');
    return match ? (
      <SyntaxHighlighter
        children={String(children).replace(/\n$/, '')}
        style={vs}
        language={match[1]}
        PreTag="div"
        className="rounded-sm border border-mountain-grey"
      />
    ) : (
      <code {...props} className="bg-mountain-grey bg-opacity-30 px-1 py-0.5 rounded-sm font-sans text-ink">
        {children}
      </code>
    );
  },
};

const ArticleEditor: React.FC = () => {
  const { id } = useParams();
  const navigate = useNavigate();
  const language = usePreferenceStore((state) => state.language);
  const isEdit = id && id !== 'new';
  const t = (key: Parameters<typeof translate>[1]) => translate(language, key);

  const [title, setTitle] = useState('');
  const [slug, setSlug] = useState('');
  const [summary, setSummary] = useState('');
  const [content, setContent] = useState('');
  const [coverUrl, setCoverUrl] = useState('');
  const [status, setStatus] = useState('draft');
  const [categoryId, setCategoryId] = useState<number | ''>('');
  const [tagIds, setTagIds] = useState<number[]>([]);

  const [categories, setCategories] = useState<Category[]>([]);
  const [tags, setTags] = useState<Tag[]>([]);
  const [error, setError] = useState('');
  const [submitting, setSubmitting] = useState(false);

  useEffect(() => {
    const fetchDeps = async () => {
      try {
        const [catRes, tagRes] = await Promise.all([
          getCategories({ size: 100 }),
          getTags({ size: 100 }),
        ]);
        setCategories(catRes.data.items || []);
        setTags(tagRes.data.items || []);
      } catch (e) {
        setError(getErrorMessage(e, translate(language, 'articleEditor.depsError')));
      }
    };
    fetchDeps();

    if (isEdit) {
      getArticleById(id)
        .then(res => {
          const article = res.data;
          setTitle(article.title);
          setSlug(article.slug);
          setSummary(article.summary);
          setContent(article.content || '');
          setCoverUrl(article.coverUrl || '');
          setStatus(article.status);
          setCategoryId(article.categoryId || '');
          if (article.tags) setTagIds(article.tags.map(tag => tag.id));
        })
        .catch(e => setError(getErrorMessage(e, translate(language, 'article.loadError'))));
    }
  }, [id, isEdit, language]);

  const handleSave = async () => {
    const payload = {
      title, slug, summary, content, coverUrl, status,
      categoryId: categoryId === '' ? 0 : Number(categoryId),
      tagIds,
    };

    setError('');
    setSubmitting(true);
    try {
      if (isEdit) {
        await updateArticle(id, payload);
      } else {
        await createArticle(payload);
      }
      navigate('/admin/articles');
    } catch (e: unknown) {
      setError(getErrorMessage(e, t('articleEditor.saveError')));
    } finally {
      setSubmitting(false);
    }
  };

  const handleTagToggle = (tagId: number) => {
    setTagIds(prev =>
      prev.includes(tagId) ? prev.filter(prevId => prevId !== tagId) : [...prev, tagId],
    );
  };

  return (
    <div className="flex flex-col h-[80vh]">
      <div className="flex justify-between items-center mb-6 pb-4 border-b border-mountain-grey">
        <h3 className="text-2xl font-bold text-ink tracking-widest">{isEdit ? t('articleEditor.editTitle') : t('articleEditor.newTitle')}</h3>
        <button onClick={handleSave} disabled={submitting} className="px-6 py-2 bg-ink text-paper tracking-widest text-sm hover:bg-opacity-80 transition-colors disabled:opacity-50 disabled:cursor-not-allowed">
          {submitting ? t('common.saving') : t('articleEditor.save')}
        </button>
      </div>

      <InlineNotice message={error} className="mb-6" />

      <div className="grid grid-cols-1 md:grid-cols-4 gap-6 mb-6">
        <input
          type="text" placeholder={t('articleEditor.titlePlaceholder')} value={title} onChange={e => setTitle(e.target.value)}
          className="w-full bg-transparent border-b border-mountain-grey py-2 text-ink focus:outline-none focus:border-ochre text-lg font-bold"
        />
        <input
          type="text" placeholder={t('articleEditor.slugPlaceholder')} value={slug} onChange={e => setSlug(e.target.value)}
          className="w-full bg-transparent border-b border-mountain-grey py-2 text-ink focus:outline-none focus:border-ochre"
        />
        <input
          type="text" placeholder={t('articleEditor.coverPlaceholder')} value={coverUrl} onChange={e => setCoverUrl(e.target.value)}
          className="w-full bg-transparent border-b border-mountain-grey py-2 text-ink focus:outline-none focus:border-ochre"
        />
        <select
          value={status} onChange={e => setStatus(e.target.value)}
          className="w-full bg-transparent border-b border-mountain-grey py-2 text-ink focus:outline-none focus:border-ochre"
        >
          <option value="draft">{getArticleStatusLabel(language, 'draft')}</option>
          <option value="published">{getArticleStatusLabel(language, 'published')}</option>
          <option value="archived">{getArticleStatusLabel(language, 'archived')}</option>
        </select>
      </div>

      <div className="grid grid-cols-1 md:grid-cols-2 gap-6 mb-6">
        <select
          value={categoryId} onChange={e => setCategoryId(e.target.value === '' ? '' : Number(e.target.value))}
          className="w-full bg-transparent border-b border-mountain-grey py-2 text-ink focus:outline-none focus:border-ochre"
        >
          <option value="">{t('articleEditor.chooseCategory')}</option>
          {categories.map(category => <option key={category.id} value={category.id}>{category.name}</option>)}
        </select>

        <div className="flex items-center space-x-2 overflow-x-auto pb-2 border-b border-mountain-grey">
          <span className="text-ink-light text-sm whitespace-nowrap">{t('articleEditor.tags')}</span>
          {tags.map(tag => (
            <span
              key={tag.id}
              onClick={() => handleTagToggle(tag.id)}
              className={`cursor-pointer px-2 py-1 text-xs border rounded-sm whitespace-nowrap ${tagIds.includes(tag.id) ? 'border-ochre text-ochre' : 'border-mountain-grey text-ink-light'}`}
            >
              {tag.name}
            </span>
          ))}
        </div>
      </div>

      <input
        type="text" placeholder={t('articleEditor.summary')} value={summary} onChange={e => setSummary(e.target.value)}
        className="w-full bg-transparent border-b border-mountain-grey py-2 text-ink focus:outline-none focus:border-ochre mb-6"
      />

      <div className="flex-grow flex flex-col md:flex-row gap-6 overflow-hidden">
        <div className="w-full md:w-1/2 flex flex-col border border-mountain-grey p-4">
          <textarea
            value={content}
            onChange={e => setContent(e.target.value)}
            className="w-full h-full bg-transparent resize-none focus:outline-none text-ink-light font-serif leading-relaxed"
            placeholder={t('articleEditor.content')}
          ></textarea>
        </div>
        <div className="w-full md:w-1/2 border border-mountain-grey p-4 overflow-y-auto bg-[var(--paper-soft)]">
          <div className="prose prose-stone max-w-none
            prose-headings:font-serif prose-headings:font-bold prose-headings:text-ink
            prose-p:text-ink-light prose-p:leading-loose prose-p:tracking-wide
            prose-a:text-ochre
            prose-blockquote:border-l-4 prose-blockquote:border-mountain-grey prose-blockquote:pl-6 prose-blockquote:italic
          ">
            <ReactMarkdown components={markdownComponents}>
              {content}
            </ReactMarkdown>
          </div>
        </div>
      </div>
    </div>
  );
};

export default ArticleEditor;
