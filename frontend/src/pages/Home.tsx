import React, { useEffect, useState } from 'react';
import { Link } from 'react-router-dom';
import Pagination from '../components/Pagination';
import http from '../utils/http';
import InlineNotice from '../components/InlineNotice';
import { getErrorMessage } from '../utils/error';

interface Article {
  id: number;
  title: string;
  slug: string;
  summary: string;
  coverUrl?: string;
  createdAt: string;
  viewCount: number;
  category?: { name: string };
  tags?: { name: string }[];
}

const Home: React.FC = () => {
  const [articles, setArticles] = useState<Article[]>([]);
  const [loading, setLoading] = useState(true);
  const [page, setPage] = useState(1);
  const [total, setTotal] = useState(0);
  const [error, setError] = useState('');
  const size = 10;

  useEffect(() => {
    const fetchArticles = async () => {
      setLoading(true);
      setError('');
      try {
        const res: any = await http.get('/articles', { params: { status: 'published', page, size } });
        setArticles(res.data.items || []);
        setTotal(res.data.total || 0);
      } catch (err) {
        setError(getErrorMessage(err, '文章列表加载失败'));
      } finally {
        setLoading(false);
      }
    };
    fetchArticles();
  }, [page]);

  if (loading) {
    return <div className="flex-grow flex items-center justify-center text-ink-light tracking-widest">研墨中...</div>;
  }

  return (
    <div className="space-y-20 mt-8 max-w-4xl mx-auto w-full">
      <InlineNotice message={error} />
      {articles.length === 0 ? (
        <div className="text-center text-ink-light italic">尚无诗篇。</div>
      ) : (
        <>
          {articles.map((article) => (
            <article key={article.id} className="group relative flex flex-col md:flex-row gap-8 items-start">
              {article.coverUrl && (
                <div className="w-full md:w-1/3 shrink-0 h-48 overflow-hidden relative">
                  <Link to={`/article/${article.id}`}>
                    <img src={article.coverUrl} alt="cover" className="w-full h-full object-cover grayscale opacity-80 group-hover:grayscale-0 group-hover:opacity-100 transition-all duration-700" />
                  </Link>
                  <div className="absolute inset-0 bg-paper bg-opacity-10 pointer-events-none"></div>
                </div>
              )}
              <div className="flex-1">
                <Link to={`/article/${article.id}`} className="block">
                  <h2 className="text-3xl font-bold text-ink mb-4 group-hover:text-ochre transition-colors duration-500">
                    {article.title}
                  </h2>
                  <p className="text-ink-light leading-relaxed mb-6 whitespace-pre-line line-clamp-3">
                    {article.summary}
                  </p>
                </Link>
                <div className="flex flex-wrap items-center gap-4 text-xs text-ink-light opacity-70 tracking-wider">
                  <span>{new Date(article.createdAt).toLocaleDateString('zh-CN', { year: 'numeric', month: 'long', day: 'numeric' })}</span>
                  <span>阅 {article.viewCount}</span>
                  {article.category && (
                    <span className="px-2 py-0.5 border border-mountain-grey border-opacity-50 group-hover:border-ochre transition-colors">
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
              {/* 水墨风分割线 */}
              <div className="absolute -bottom-10 left-0 w-24 h-px bg-mountain-grey opacity-50 group-hover:w-full group-hover:bg-ochre transition-all duration-700 ease-in-out"></div>
            </article>
          ))}
          
          <Pagination 
            currentPage={page} 
            total={total} 
            pageSize={size} 
            onPageChange={setPage} 
          />
        </>
      )}
    </div>
  );
};

export default Home;
