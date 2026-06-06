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

const markdownComponents: Components = {
  code({ className, children, ...props }) {
    const match = /language-(\w+)/.exec(className || '')
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
    )
  }
};

const ArticleEditor: React.FC = () => {
  const { id } = useParams();
  const navigate = useNavigate();
  const isEdit = id && id !== 'new';

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
          getTags({ size: 100 })
        ]);
        setCategories(catRes.data.items || []);
        setTags(tagRes.data.items || []);
      } catch (e) {
        setError(getErrorMessage(e, '分类和标签加载失败'));
      }
    };
    fetchDeps();

    if (isEdit) {
      getArticleById(id)
        .then(res => {
          const a = res.data;
          setTitle(a.title);
          setSlug(a.slug);
          setSummary(a.summary);
          setContent(a.content || '');
          setCoverUrl(a.coverUrl || '');
          setStatus(a.status);
          setCategoryId(a.categoryId || '');
          if (a.tags) setTagIds(a.tags.map(t => t.id));
        })
        .catch(e => setError(getErrorMessage(e, '文章加载失败')));
    }
  }, [id, isEdit]);

  const handleSave = async () => {
    const payload = {
      title, slug, summary, content, coverUrl, status,
      categoryId: categoryId === '' ? 0 : Number(categoryId),
      tagIds
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
      setError(getErrorMessage(e, '保存失败'));
    } finally {
      setSubmitting(false);
    }
  };

  const handleTagToggle = (tagId: number) => {
    setTagIds(prev => 
      prev.includes(tagId) ? prev.filter(id => id !== tagId) : [...prev, tagId]
    );
  };

  return (
    <div className="flex flex-col h-[80vh]">
      {/* Editor Header */}
      <div className="flex justify-between items-center mb-6 pb-4 border-b border-mountain-grey">
        <h3 className="text-2xl font-bold text-ink tracking-widest">{isEdit ? '修编' : '提笔'}</h3>
        <button onClick={handleSave} disabled={submitting} className="px-6 py-2 bg-ink text-paper tracking-widest text-sm hover:bg-opacity-80 transition-colors disabled:opacity-50 disabled:cursor-not-allowed">
          {submitting ? '保存中...' : '落笔保存'}
        </button>
      </div>

      <InlineNotice message={error} className="mb-6" />

      {/* Meta Infos */}
      <div className="grid grid-cols-1 md:grid-cols-4 gap-6 mb-6">
        <input 
          type="text" placeholder="标题" value={title} onChange={e => setTitle(e.target.value)}
          className="w-full bg-transparent border-b border-mountain-grey py-2 text-ink focus:outline-none focus:border-ochre text-lg font-bold"
        />
        <input 
          type="text" placeholder="Slug (路径)" value={slug} onChange={e => setSlug(e.target.value)}
          className="w-full bg-transparent border-b border-mountain-grey py-2 text-ink focus:outline-none focus:border-ochre"
        />
        <input 
          type="text" placeholder="封面 URL" value={coverUrl} onChange={e => setCoverUrl(e.target.value)}
          className="w-full bg-transparent border-b border-mountain-grey py-2 text-ink focus:outline-none focus:border-ochre"
        />
        <select 
          value={status} onChange={e => setStatus(e.target.value)}
          className="w-full bg-transparent border-b border-mountain-grey py-2 text-ink focus:outline-none focus:border-ochre"
        >
          <option value="draft">草稿</option>
          <option value="published">发布</option>
          <option value="archived">归档</option>
        </select>
      </div>

      <div className="grid grid-cols-1 md:grid-cols-2 gap-6 mb-6">
        <select 
          value={categoryId} onChange={e => setCategoryId(Number(e.target.value))}
          className="w-full bg-transparent border-b border-mountain-grey py-2 text-ink focus:outline-none focus:border-ochre"
        >
          <option value="">选择分类...</option>
          {categories.map(c => <option key={c.id} value={c.id}>{c.name}</option>)}
        </select>
        
        <div className="flex items-center space-x-2 overflow-x-auto pb-2 border-b border-mountain-grey">
          <span className="text-ink-light text-sm whitespace-nowrap">标签:</span>
          {tags.map(t => (
            <span 
              key={t.id} 
              onClick={() => handleTagToggle(t.id)}
              className={`cursor-pointer px-2 py-1 text-xs border rounded-sm whitespace-nowrap ${tagIds.includes(t.id) ? 'border-ochre text-ochre' : 'border-mountain-grey text-ink-light'}`}
            >
              {t.name}
            </span>
          ))}
        </div>
      </div>

      <input 
        type="text" placeholder="摘要" value={summary} onChange={e => setSummary(e.target.value)}
        className="w-full bg-transparent border-b border-mountain-grey py-2 text-ink focus:outline-none focus:border-ochre mb-6"
      />

      {/* Split Screen */}
      <div className="flex-grow flex flex-col md:flex-row gap-6 overflow-hidden">
        {/* Left: Textarea */}
        <div className="w-full md:w-1/2 flex flex-col border border-mountain-grey p-4">
          <textarea 
            value={content}
            onChange={e => setContent(e.target.value)}
            className="w-full h-full bg-transparent resize-none focus:outline-none text-ink-light font-serif leading-relaxed"
            placeholder="正文 (Markdown)..."
          ></textarea>
        </div>
        {/* Right: Preview */}
        <div className="w-full md:w-1/2 border border-mountain-grey p-4 overflow-y-auto bg-white bg-opacity-50">
          <div className="prose prose-stone max-w-none
            prose-headings:font-serif prose-headings:font-bold prose-headings:text-ink
            prose-p:text-ink-light prose-p:leading-loose prose-p:tracking-wide
            prose-a:text-ochre
            prose-blockquote:border-l-4 prose-blockquote:border-mountain-grey prose-blockquote:pl-6 prose-blockquote:italic
          ">
            <ReactMarkdown
              components={markdownComponents}
            >
              {content}
            </ReactMarkdown>
          </div>
        </div>
      </div>
    </div>
  );
};

export default ArticleEditor;
