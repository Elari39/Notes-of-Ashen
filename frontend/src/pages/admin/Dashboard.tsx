import React, { useEffect, useState } from 'react';
import { Link } from 'react-router-dom';
import {
  Bar,
  BarChart,
  CartesianGrid,
  Legend,
  Line,
  LineChart,
  ResponsiveContainer,
  Tooltip,
  XAxis,
  YAxis,
} from 'recharts';
import { getAdminStats } from '../../api/admin';
import InlineNotice from '../../components/InlineNotice';
import PagePendingState from '../../components/RoutePending';
import { getArticleStatusLabel, getDateLocale, translate } from '../../i18n';
import { usePreferenceStore, type Language } from '../../store/preferences';
import { getErrorMessage } from '../../utils/error';
import type { AdminStats, Article, GeoStat, Log, RefererStat, TrafficTrendPoint } from '../../types';

const statItems = [
  'articleTotal',
  'publishedTotal',
  'draftTotal',
  'scheduledTotal',
  'viewTotal',
  'likeTotal',
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
    let active = true;
    const fetchStats = async () => {
      setLoading(true);
      setError('');
      try {
        const res = await getAdminStats();
        if (active) {
          setStats(res.data);
        }
      } catch (e) {
        if (active) {
          setError(getErrorMessage(e, translate(language, 'dashboard.loadError')));
        }
      } finally {
        if (active) {
          setLoading(false);
        }
      }
    };
    fetchStats();
    return () => {
      active = false;
    };
  }, [language]);

  if (loading && !stats) {
    return <PagePendingState variant="admin" label={t('dashboard.loading')} />;
  }

  return (
    <div>
      <div className="mb-8 border-b border-mountain-grey pb-4">
        <h3 className="text-2xl font-bold tracking-widest text-ink">{t('dashboard.title')}</h3>
      </div>

      <InlineNotice message={error} className="mb-6" />
      {loading && <PagePendingState variant="inline" label={t('dashboard.loading')} />}

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

          <TrafficOverview
            trend={stats.trafficTrend || []}
            referers={stats.topReferers || []}
            todayPv={stats.todayPv || 0}
            todayUv={stats.todayUv || 0}
            language={language}
          />

          <GeoOverview geoStats={stats.geoStats || []} language={language} />

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

const GeoOverview: React.FC<{
  geoStats: GeoStat[];
  language: Language;
}> = ({ geoStats, language }) => {
  const labels = dashboardGeoLabels(language);
  const chartData = geoStats.map((item) => ({
    ...item,
    label: geoLabel(item, language),
  }));

  return (
    <section>
      <div className="mb-4">
        <h4 className="text-sm font-bold tracking-widest text-ink">{labels.title}</h4>
        <p className="mt-2 text-xs text-ink-light">{labels.subtitle}</p>
      </div>
      <div className="grid gap-6 lg:grid-cols-[minmax(0,2fr)_minmax(16rem,1fr)]">
        <div className="h-72 border border-mountain-grey bg-[var(--paper-soft)] p-3">
          {chartData.length === 0 ? (
            <div className="flex h-full items-center justify-center text-sm text-ink-light">{labels.empty}</div>
          ) : (
            <ResponsiveContainer width="100%" height="100%">
              <BarChart data={chartData} margin={{ top: 12, right: 12, bottom: 0, left: -18 }}>
                <CartesianGrid stroke="var(--mountain-grey)" strokeDasharray="3 3" />
                <XAxis dataKey="label" tick={{ fill: 'var(--ink-light)', fontSize: 11 }} />
                <YAxis tick={{ fill: 'var(--ink-light)', fontSize: 11 }} allowDecimals={false} />
                <Tooltip
                  contentStyle={{
                    background: 'var(--paper)',
                    border: '1px solid var(--mountain-grey)',
                    borderRadius: 4,
                    color: 'var(--ink)',
                  }}
                />
                <Legend />
                <Bar dataKey="pv" name="PV" fill="var(--ochre)" />
                <Bar dataKey="uv" name="UV" fill="var(--code-blue)" />
              </BarChart>
            </ResponsiveContainer>
          )}
        </div>
        <div className="border border-mountain-grey bg-[var(--paper-soft)] p-4">
          <h5 className="mb-4 text-xs font-bold tracking-widest text-ink">{labels.locations}</h5>
          {geoStats.length === 0 ? (
            <p className="text-sm text-ink-light">{labels.empty}</p>
          ) : (
            <div className="space-y-3">
              {geoStats.map((item) => (
                <div key={`${item.countryCode}:${item.regionName}:${item.cityName}`} className="flex items-center justify-between gap-3 text-sm">
                  <span className="min-w-0 truncate text-ink">{geoLabel(item, language)}</span>
                  <span className="shrink-0 text-ink-light">PV {item.pv} / UV {item.uv}</span>
                </div>
              ))}
            </div>
          )}
        </div>
      </div>
    </section>
  );
};

const TrafficOverview: React.FC<{
  trend: TrafficTrendPoint[];
  referers: RefererStat[];
  todayPv: number;
  todayUv: number;
  language: Language;
}> = ({ trend, referers, todayPv, todayUv, language }) => {
  const labels = dashboardTrafficLabels(language);
  const sourceTotal = referers.reduce((sum, item) => sum + item.pv, 0);

  return (
    <section>
      <div className="mb-4 flex flex-col gap-3 sm:flex-row sm:items-end sm:justify-between">
        <div>
          <h4 className="text-sm font-bold tracking-widest text-ink">{labels.title}</h4>
          <p className="mt-2 text-xs text-ink-light">{labels.subtitle}</p>
        </div>
        <div className="grid grid-cols-2 gap-2 text-sm">
          <div className="border border-mountain-grey px-3 py-2">
            <span className="block text-xs text-ink-light">{labels.todayPv}</span>
            <strong className="mt-1 block text-xl text-ink">{todayPv}</strong>
          </div>
          <div className="border border-mountain-grey px-3 py-2">
            <span className="block text-xs text-ink-light">{labels.todayUv}</span>
            <strong className="mt-1 block text-xl text-ink">{todayUv}</strong>
          </div>
        </div>
      </div>

      <div className="grid gap-6 lg:grid-cols-[minmax(0,2fr)_minmax(16rem,1fr)]">
        <div className="h-72 border border-mountain-grey bg-[var(--paper-soft)] p-3">
          <ResponsiveContainer width="100%" height="100%">
            <LineChart data={trend} margin={{ top: 12, right: 12, bottom: 0, left: -18 }}>
              <CartesianGrid stroke="var(--mountain-grey)" strokeDasharray="3 3" />
              <XAxis dataKey="date" tickFormatter={(value) => shortDate(value, language)} tick={{ fill: 'var(--ink-light)', fontSize: 11 }} />
              <YAxis tick={{ fill: 'var(--ink-light)', fontSize: 11 }} allowDecimals={false} />
              <Tooltip
                labelFormatter={(value) => fullDate(String(value), language)}
                contentStyle={{
                  background: 'var(--paper)',
                  border: '1px solid var(--mountain-grey)',
                  borderRadius: 4,
                  color: 'var(--ink)',
                }}
              />
              <Legend />
              <Line type="monotone" dataKey="pv" name="PV" stroke="var(--ochre)" strokeWidth={2} dot={false} activeDot={{ r: 4 }} />
              <Line type="monotone" dataKey="uv" name="UV" stroke="var(--code-blue)" strokeWidth={2} dot={false} activeDot={{ r: 4 }} />
            </LineChart>
          </ResponsiveContainer>
        </div>

        <div className="border border-mountain-grey bg-[var(--paper-soft)] p-4">
          <h5 className="mb-4 text-xs font-bold tracking-widest text-ink">{labels.referers}</h5>
          {referers.length === 0 ? (
            <p className="text-sm text-ink-light">{labels.empty}</p>
          ) : (
            <div className="space-y-3">
              {referers.map((item) => (
                <div key={`${item.sourceType}:${item.sourceName}`}>
                  <div className="mb-1 flex items-center justify-between gap-3 text-sm">
                    <span className="min-w-0 truncate text-ink">{sourceLabel(item, language)}</span>
                    <span className="shrink-0 text-ink-light">{item.pv}</span>
                  </div>
                  <div className="h-1.5 overflow-hidden bg-[var(--mountain-grey)]">
                    <div
                      className="h-full bg-[var(--ochre)]"
                      style={{ width: `${sourceTotal > 0 ? Math.max(4, (item.pv / sourceTotal) * 100) : 0}%` }}
                    />
                  </div>
                </div>
              ))}
            </div>
          )}
        </div>
      </div>
    </section>
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

const dashboardTrafficLabels = (language: Language) => language === 'zh'
  ? {
      title: '访问趋势',
      subtitle: '过去 30 天公开页面 PV / UV',
      todayPv: '今日 PV',
      todayUv: '今日 UV',
      referers: '访问来源',
      empty: '暂无访问数据',
      direct: '直接访问',
      internal: '站内跳转',
      search: '搜索引擎',
      external: '外部网站',
    }
  : {
      title: 'Traffic Trend',
      subtitle: 'Public page PV / UV over the last 30 days',
      todayPv: 'Today PV',
      todayUv: 'Today UV',
      referers: 'Referers',
      empty: 'No traffic data yet',
      direct: 'Direct',
      internal: 'Internal',
      search: 'Search',
      external: 'External',
    };

const dashboardGeoLabels = (language: Language) => language === 'zh'
  ? {
      title: '访客地理分布',
      subtitle: '过去 30 天按国家 / 城市聚合的访问分布',
      locations: '热门地区',
      empty: '暂无地理位置数据',
    }
  : {
      title: 'Visitor Geography',
      subtitle: 'Country and city distribution over the last 30 days',
      locations: 'Top Locations',
      empty: 'No geo data yet',
    };

const geoLabel = (item: GeoStat, language: Language) => [item.countryName, item.regionName, item.cityName]
  .filter((value) => value && value !== 'Unknown')
  .join(' / ') || (language === 'zh' ? '未知地区' : 'Unknown');

const shortDate = (value: string, language: Language) => {
  const date = new Date(`${value}T00:00:00`);
  if (Number.isNaN(date.getTime())) {
    return value;
  }
  return date.toLocaleDateString(getDateLocale(language), { month: '2-digit', day: '2-digit' });
};

const fullDate = (value: string, language: Language) => {
  const date = new Date(`${value}T00:00:00`);
  if (Number.isNaN(date.getTime())) {
    return value;
  }
  return date.toLocaleDateString(getDateLocale(language));
};

const sourceLabel = (item: RefererStat, language: Language) => {
  const labels = dashboardTrafficLabels(language);
  if (item.sourceType === 'direct') return labels.direct;
  if (item.sourceType === 'internal') return labels.internal;
  if (item.sourceType === 'search') return `${labels.search}: ${item.sourceName}`;
  if (item.sourceType === 'external') return `${labels.external}: ${item.sourceName}`;
  return item.sourceName;
};

export default AdminDashboard;
