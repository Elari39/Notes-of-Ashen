import React, { useEffect, useState } from 'react';
import { Link } from 'react-router-dom';
import { getAdminStats } from '../../api/admin';
import InlineNotice from '../../components/InlineNotice';
import { getArticleStatusLabel, getDateLocale, translate } from '../../i18n';
import { usePreferenceStore, type Language } from '../../store/preferences';
import { getErrorMessage } from '../../utils/error';
import type { AdminStats, Article, Log } from '../../types';

const statItems = [
  'articleTotal',
  'publishedTotal',
  'draftTotal',
  'scheduledTotal',
  'viewTotal',
  'userTotal',
  'categoryTotal',
  'tagTotal',
] as const;

const AdminDashboard: React.FC = () => {
  const language = usePreferenceStore((state) => state.language);
  const [stats, setStats] = useState<AdminStats | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');
  const t = (key: Parameters<typeof translate>[1]) => translate(language, key);

  useEffect(() => {
    const fetchStats = async () => {
      setLoading(true);
      setError('');
      try {
        const res = await getAdminStats();
        setStats(res.data);
      } catch (e) {
        setError(getErrorMessage(e, translate(language, 'dashboard.loadError')));
      } finally {
        setLoading(false);
      }
    };
    fetchStats();
  }, [language]);

  if (loading) {
    return <div className="py-20 text-center tracking-widest text-ink-light">{t('dashboard.loading')}</div>;
  }

  return (
    <div>
      <div className="mb-8 border-b border-mountain-grey pb-4">
        <h3 className="text-2xl font-bold tracking-widest text-ink">{t('dashboard.title')}</h3>
      </div>

      <InlineNotice message={error} className="mb-6" />

      {stats && (
        <div className="space-y-10">
          <div className="grid grid-cols-2 gap-3 md:grid-cols-4">
            {statItems.map((key) => (
              <div key={key} className="border border-mountain-grey bg-[var(--paper-soft)] p-4">
                <p className="text-xs tracking-widest text-ink-light">{t(`dashboard.${key}`)}</p>
                <p className="mt-3 text-3xl font-bold text-ink">{stats[key]}</p>
              </div>
            ))}
          </div>

          <div className="grid gap-8 lg:grid-cols-2">
            <ArticleList
              title={t('dashboard.popularArticles')}
              articles={stats.popularArticles}
              metric="views"
              language={language}
              emptyLabel={t('dashboard.empty')}
              viewsLabel={t('common.views')}
            />
            <ArticleList
              title={t('dashboard.recentArticles')}
              articles={stats.recentArticles}
              metric="status"
              language={language}
              emptyLabel={t('dashboard.empty')}
              viewsLabel={t('common.views')}
            />
          </div>

          <section>
            <h4 className="mb-4 text-sm font-bold tracking-widest text-ink">{t('dashboard.recentLogs')}</h4>
            <div className="border border-mountain-grey">
              {stats.recentLogs.length === 0 && <p className="p-4 text-sm text-ink-light">{t('dashboard.empty')}</p>}
              {stats.recentLogs.map((log) => (
                <LogRow key={log.id} log={log} language={language} />
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
  language: Language;
  emptyLabel: string;
  viewsLabel: string;
}> = ({ title, articles, metric, language, emptyLabel, viewsLabel }) => (
  <section>
    <h4 className="mb-4 text-sm font-bold tracking-widest text-ink">{title}</h4>
    <div className="space-y-3">
      {articles.length === 0 && <p className="text-sm text-ink-light">{emptyLabel}</p>}
      {articles.map((article) => (
        <Link key={article.id} to={`/admin/editor/${article.id}`} className="block border-b border-mountain-grey py-3 hover:text-ochre">
          <span className="font-bold">{article.title}</span>
          <span className="ml-3 text-xs text-ink-light">
            {metric === 'views' ? `${article.viewCount} ${viewsLabel}` : getArticleStatusLabel(language, article.status)}
          </span>
        </Link>
      ))}
    </div>
  </section>
);

const LogRow: React.FC<{ log: Log; language: Language }> = ({ log, language }) => (
  <div className="grid gap-2 border-b border-mountain-grey p-4 text-sm last:border-b-0 md:grid-cols-[1fr_auto] md:items-center">
    <div>
      <p className="font-bold text-ink">{log.eventType}</p>
      <p className="mt-1 text-xs text-ink-light">
        {log.resourceType}
        {log.resourceId ? ` #${log.resourceId}` : ''}{log.ip ? ` · ${log.ip}` : ''}
      </p>
    </div>
    <time className="text-xs text-ink-light">{new Date(log.createdAt).toLocaleString(getDateLocale(language))}</time>
  </div>
);

export default AdminDashboard;
