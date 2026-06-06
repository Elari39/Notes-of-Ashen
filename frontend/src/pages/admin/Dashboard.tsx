import React, { useEffect, useState } from 'react';
import { Link } from 'react-router-dom';
import { getAdminStats } from '../../api/admin';
import InlineNotice from '../../components/InlineNotice';
import { getErrorMessage } from '../../utils/error';
import type { AdminStats, Article, Log } from '../../types';

const statItems = [
  ['articleTotal', '文章总数'],
  ['publishedTotal', '已发布'],
  ['draftTotal', '草稿'],
  ['scheduledTotal', '定时发布'],
  ['viewTotal', '阅读量'],
  ['userTotal', '用户'],
  ['categoryTotal', '分类'],
  ['tagTotal', '标签'],
] as const;

const AdminDashboard: React.FC = () => {
  const [stats, setStats] = useState<AdminStats | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');

  useEffect(() => {
    const fetchStats = async () => {
      setLoading(true);
      setError('');
      try {
        const res = await getAdminStats();
        setStats(res.data);
      } catch (e) {
        setError(getErrorMessage(e, '统计数据加载失败'));
      } finally {
        setLoading(false);
      }
    };
    fetchStats();
  }, []);

  if (loading) {
    return <div className="py-20 text-center tracking-widest text-ink-light">统计加载中...</div>;
  }

  return (
    <div>
      <div className="mb-8 border-b border-mountain-grey pb-4">
        <h3 className="text-2xl font-bold tracking-widest text-ink">数据统计</h3>
      </div>

      <InlineNotice message={error} className="mb-6" />

      {stats && (
        <div className="space-y-10">
          <div className="grid grid-cols-2 gap-3 md:grid-cols-4">
            {statItems.map(([key, label]) => (
              <div key={key} className="border border-mountain-grey bg-[var(--paper-soft)] p-4">
                <p className="text-xs tracking-widest text-ink-light">{label}</p>
                <p className="mt-3 text-3xl font-bold text-ink">{stats[key]}</p>
              </div>
            ))}
          </div>

          <div className="grid gap-8 lg:grid-cols-2">
            <ArticleList title="热门文章" articles={stats.popularArticles} metric="views" />
            <ArticleList title="最近文章" articles={stats.recentArticles} metric="status" />
          </div>

          <section>
            <h4 className="mb-4 text-sm font-bold tracking-widest text-ink">最近日志</h4>
            <div className="border border-mountain-grey">
              {stats.recentLogs.length === 0 && <p className="p-4 text-sm text-ink-light">暂无数据</p>}
              {stats.recentLogs.map((log) => (
                <LogRow key={log.id} log={log} />
              ))}
            </div>
          </section>
        </div>
      )}
    </div>
  );
};

const ArticleList: React.FC<{
  title: string;
  articles: Article[];
  metric: 'views' | 'status';
}> = ({ title, articles, metric }) => (
  <section>
    <h4 className="mb-4 text-sm font-bold tracking-widest text-ink">{title}</h4>
    <div className="space-y-3">
      {articles.length === 0 && <p className="text-sm text-ink-light">暂无数据</p>}
      {articles.map((article) => (
        <Link key={article.id} to={`/admin/editor/${article.id}`} className="block border-b border-mountain-grey py-3 hover:text-ochre">
          <span className="font-bold">{article.title}</span>
          <span className="ml-3 text-xs text-ink-light">
            {metric === 'views' ? `${article.viewCount} views` : article.status}
          </span>
        </Link>
      ))}
    </div>
  </section>
);

const LogRow: React.FC<{ log: Log }> = ({ log }) => (
  <div className="grid gap-2 border-b border-mountain-grey p-4 text-sm last:border-b-0 md:grid-cols-[1fr_auto] md:items-center">
    <div>
      <p className="font-bold text-ink">{log.eventType}</p>
      <p className="mt-1 text-xs text-ink-light">
        {log.resourceType}
        {log.resourceId ? ` #${log.resourceId}` : ''}{log.ip ? ` · ${log.ip}` : ''}
      </p>
    </div>
    <time className="text-xs text-ink-light">{new Date(log.createdAt).toLocaleString()}</time>
  </div>
);

export default AdminDashboard;
