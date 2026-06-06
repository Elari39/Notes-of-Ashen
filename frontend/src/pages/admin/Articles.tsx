import React, { useCallback, useEffect, useState } from 'react';
import { getArticles, updateArticleStatus, deleteArticle } from '../../api/article';
import { Article } from '../../types';
import { Link, useNavigate } from 'react-router-dom';
import Pagination from '../../components/Pagination';
import InlineNotice from '../../components/InlineNotice';
import { getErrorMessage } from '../../utils/error';

const AdminArticles: React.FC = () => {
  const [articles, setArticles] = useState<Article[]>([]);
  const [page, setPage] = useState(1);
  const [total, setTotal] = useState(0);
  const [error, setError] = useState('');
  const [busyId, setBusyId] = useState<number | null>(null);
  const size = 10;
  const navigate = useNavigate();

  const fetchList = useCallback(async () => {
    try {
      const res = await getArticles({ page, size });
      setArticles(res.data.items || []);
      setTotal(res.data.total || 0);
    } catch (e) {
      setError(getErrorMessage(e, '文章列表加载失败'));
    }
  }, [page, size]);

  useEffect(() => {
    fetchList();
  }, [fetchList]);

  const handleStatus = async (id: number, status: string) => {
    setError('');
    setBusyId(id);
    try {
      await updateArticleStatus(id, status);
      fetchList();
    } catch (e) {
      setError(getErrorMessage(e, '状态更新失败'));
    } finally {
      setBusyId(null);
    }
  };

  const handleDelete = async (id: number) => {
    if (confirm('确认删除此篇？')) {
      setError('');
      setBusyId(id);
      try {
        await deleteArticle(id);
        fetchList();
      } catch (e) {
        setError(getErrorMessage(e, '删除失败'));
      } finally {
        setBusyId(null);
      }
    }
  };

  return (
    <div>
      <div className="flex justify-between items-center mb-8 border-b border-mountain-grey pb-4">
        <h3 className="text-2xl font-bold text-ink tracking-widest">文章管理</h3>
        <button onClick={() => navigate('/admin/editor/new')} className="px-4 py-2 border border-ink text-ink hover:bg-ink hover:text-paper tracking-widest text-sm transition-colors">
          提笔
        </button>
      </div>

      <InlineNotice message={error} className="mb-6" />
      
      <table className="w-full text-left border-collapse text-sm">
        <thead>
          <tr className="border-b border-mountain-grey text-ink-light opacity-80 tracking-widest">
            <th className="py-3 font-normal">标题</th>
            <th className="py-3 font-normal">分类/标签</th>
            <th className="py-3 font-normal">状态</th>
            <th className="py-3 font-normal">时间</th>
            <th className="py-3 font-normal text-right">操作</th>
          </tr>
        </thead>
        <tbody>
          {articles.map(a => (
            <tr key={a.id} className="border-b border-mountain-grey border-opacity-50 hover:bg-mountain-grey hover:bg-opacity-20 transition-colors text-ink">
              <td className="py-4 font-bold">{a.title}</td>
              <td className="py-4">
                <div className="flex flex-col space-y-1">
                  {a.category && (
                    <span className="text-ochre text-xs">{a.category.name}</span>
                  )}
                  {a.tags && a.tags.length > 0 && (
                    <div className="flex space-x-2 text-xs text-ink-light opacity-80">
                      {a.tags.map(t => <span key={t.name}>#{t.name}</span>)}
                    </div>
                  )}
                </div>
              </td>
              <td className="py-4">
                <span className={`px-2 py-1 text-xs border ${a.status === 'published' ? 'border-ochre text-ochre' : 'border-ink-light text-ink-light'}`}>
                  {a.status === 'published' ? '已发布' : a.status === 'draft' ? '草稿' : '归档'}
                </span>
              </td>
              <td className="py-4 text-ink-light opacity-80">
                {new Date(a.createdAt).toLocaleDateString()}
              </td>
              <td className="py-4 text-right space-x-4 tracking-wider">
                <Link to={`/admin/editor/${a.id}`} className="hover:text-ochre">修编</Link>
                {a.status !== 'published' && (
                  <button onClick={() => handleStatus(a.id, 'published')} disabled={busyId === a.id} className="hover:text-ochre disabled:opacity-50 disabled:cursor-not-allowed">发布</button>
                )}
                {a.status === 'published' && (
                  <button onClick={() => handleStatus(a.id, 'archived')} disabled={busyId === a.id} className="hover:text-ochre disabled:opacity-50 disabled:cursor-not-allowed">归档</button>
                )}
                <button onClick={() => handleDelete(a.id)} disabled={busyId === a.id} className="text-ochre opacity-80 hover:opacity-100 disabled:opacity-50 disabled:cursor-not-allowed">销毁</button>
              </td>
            </tr>
          ))}
        </tbody>
      </table>
      <Pagination 
        currentPage={page} 
        total={total} 
        pageSize={size} 
        onPageChange={setPage} 
      />
    </div>
  );
};

export default AdminArticles;
