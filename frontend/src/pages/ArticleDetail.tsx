import React, { useEffect, useState } from 'react';
import { useParams, Link } from 'react-router-dom';
import ReactMarkdown from 'react-markdown';
import { Components } from 'react-markdown';
import { Prism as SyntaxHighlighter } from 'react-syntax-highlighter';
import { vs } from 'react-syntax-highlighter/dist/esm/styles/prism';
import http from '../utils/http';
import InlineNotice from '../components/InlineNotice';
import { getErrorMessage } from '../utils/error';
import { BaseResp } from '../types';

interface ArticleDetailData {
  id: number;
  title: string;
  content: string;
  coverUrl?: string;
  createdAt: string;
  viewCount: number;
  category?: { name: string };
  tags?: { name: string }[];
}

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

const ArticleDetail: React.FC = () => {
  const { id } = useParams<{ id: string }>();
  const [article, setArticle] = useState<ArticleDetailData | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');

  useEffect(() => {
    const fetchArticle = async () => {
      try {
        const res = await http.get<unknown, BaseResp<ArticleDetailData>>(`/articles/${id}`);
        setArticle(res.data);
      } catch (err) {
        setError(getErrorMessage(err, '文章加载失败'));
      } finally {
        setLoading(false);
      }
    };
    if (id) fetchArticle();
  }, [id]);

  if (loading) {
    return <div className="flex-grow flex items-center justify-center text-ink-light tracking-widest">展卷中...</div>;
  }

  if (!article) {
    return (
      <div className="max-w-md mx-auto mt-20">
        <InlineNotice message={error || '此卷已失。'} />
      </div>
    );
  }

  return (
    <article className="max-w-2xl mx-auto w-full">
      <header className="mb-16 text-center">
        {article.coverUrl && (
          <div className="mb-12 w-full h-64 md:h-80 overflow-hidden relative">
            <img src={article.coverUrl} alt="cover" className="w-full h-full object-cover grayscale hover:grayscale-0 transition-all duration-700" />
            <div className="absolute inset-0 bg-paper bg-opacity-20 pointer-events-none"></div>
          </div>
        )}
        <h1 className="text-4xl md:text-5xl font-bold text-ink mb-8 leading-tight">
          {article.title}
        </h1>
        <div className="flex flex-col items-center justify-center space-y-4 text-sm text-ink-light tracking-widest opacity-80">
          <div className="flex space-x-6">
            <span>{new Date(article.createdAt).toLocaleDateString('zh-CN', { year: 'numeric', month: 'long', day: 'numeric' })}</span>
            <span>阅 {article.viewCount}</span>
          </div>
          <div className="flex items-center space-x-4 text-xs">
            {article.category && (
              <span className="px-2 py-1 border border-mountain-grey text-ochre">
                {article.category.name}
              </span>
            )}
            {article.tags && article.tags.length > 0 && (
              <div className="flex space-x-3">
                {article.tags.map(t => (
                  <span key={t.name} className="relative before:content-['#'] before:mr-1 before:opacity-30">
                    {t.name}
                  </span>
                ))}
              </div>
            )}
          </div>
        </div>
      </header>

      <div className="prose prose-lg prose-stone mx-auto 
        prose-headings:font-serif prose-headings:font-bold prose-headings:text-ink
        prose-p:text-ink-light prose-p:leading-loose prose-p:tracking-wide
        prose-a:text-ochre prose-a:no-underline hover:prose-a:underline
        prose-blockquote:border-l-4 prose-blockquote:border-mountain-grey prose-blockquote:pl-6 prose-blockquote:italic prose-blockquote:text-ink-light
        prose-strong:text-ink font-serif
      ">
        <ReactMarkdown
          components={markdownComponents}
        >
          {article.content}
        </ReactMarkdown>
      </div>

      <div className="mt-20 pt-8 border-t border-mountain-grey border-opacity-50 text-center">
        <Link to="/" className="inline-block px-6 py-2 border border-ink text-ink hover:bg-ink hover:text-paper transition-colors duration-300 tracking-widest">
          合卷
        </Link>
      </div>
    </article>
  );
};

export default ArticleDetail;
