import React, { useEffect, useRef, useState } from 'react';
import {
  getAnalyticsOverview,
  getArticleAnalytics,
  getArticleAnalyticsDetail,
} from '../../api/analytics';
import type { AnalyticsOverview, ArticleAnalytics, ArticleAnalyticsDetail } from '../../types';
import { usePreferenceStore } from '../../store/preferences';
import { formatText, translate } from '../../i18n';
import { getErrorMessage } from '../../utils/error';
import { loadECharts, type ECharts } from '../../utils/loadECharts';
import InlineNotice from '../../components/InlineNotice';
import PagePendingState from '../../components/RoutePending';
import Button from '../../components/ui/Button';
import TextField from '../../components/ui/TextField';

const isoDate = (date: Date) =>
  `${date.getFullYear()}-${String(date.getMonth() + 1).padStart(2, '0')}-${String(date.getDate()).padStart(2, '0')}`;

const defaultTo = new Date();
const defaultFrom = new Date();
defaultFrom.setDate(defaultFrom.getDate() - 29);

const AdminAnalytics: React.FC = () => {
  const language = usePreferenceStore((state) => state.language);
  const t = (key: Parameters<typeof translate>[1]) => translate(language, key);
  const [from, setFrom] = useState(isoDate(defaultFrom));
  const [to, setTo] = useState(isoDate(defaultTo));
  const [applied, setApplied] = useState({ from, to });
  const [overview, setOverview] = useState<AnalyticsOverview | null>(null);
  const [articles, setArticles] = useState<ArticleAnalytics[]>([]);
  const [detail, setDetail] = useState<ArticleAnalyticsDetail | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');

  useEffect(() => {
    const controller = new AbortController();
    setLoading(true);
    setError('');
    Promise.all([
      getAnalyticsOverview(applied, controller.signal),
      getArticleAnalytics({ ...applied, page: 1, size: 20 }, controller.signal),
    ])
      .then(([summary, list]) => {
        setOverview(summary.data);
        setArticles(list.data.items || []);
      })
      .catch((err) => {
        if (!controller.signal.aborted) {
          setError(getErrorMessage(err, translate(language, 'analytics.loadError')));
        }
      })
      .finally(() => {
        if (!controller.signal.aborted) {
          setLoading(false);
        }
      });
    return () => controller.abort();
  }, [applied, language]);

  const applyPreset = (days: number) => {
    const end = new Date();
    const start = new Date();
    start.setDate(start.getDate() - (days - 1));
    const next = { from: isoDate(start), to: isoDate(end) };
    setFrom(next.from);
    setTo(next.to);
    setApplied(next);
  };

  const openDetail = async (article: ArticleAnalytics) => {
    setError('');
    try {
      const response = await getArticleAnalyticsDetail(article.articleId, applied);
      setDetail(response.data);
    } catch (err) {
      setError(getErrorMessage(err, t('analytics.loadError')));
    }
  };

  return (
    <div>
      <header className="mb-8 border-b border-hairline pb-5">
        <p className="editorial-kicker">{t('admin.analytics')}</p>
        <h3 className="mt-3 text-4xl text-ink">{t('analytics.title')}</h3>
        <p className="mt-2 text-sm text-muted">{t('analytics.subtitle')}</p>
      </header>

      <div className="mb-6 flex flex-wrap items-end gap-3">
        <div className="flex gap-2">
          <Button size="sm" variant="subtle" onClick={() => applyPreset(7)}>{t('analytics.range7')}</Button>
          <Button size="sm" variant="subtle" onClick={() => applyPreset(30)}>{t('analytics.range30')}</Button>
          <Button size="sm" variant="subtle" onClick={() => applyPreset(90)}>{t('analytics.range90')}</Button>
        </div>
        <label className="text-xs text-muted">
          {t('analytics.from')}
          <TextField type="date" value={from} onChange={(event) => setFrom(event.target.value)} />
        </label>
        <label className="text-xs text-muted">
          {t('analytics.to')}
          <TextField type="date" value={to} onChange={(event) => setTo(event.target.value)} />
        </label>
        <Button onClick={() => setApplied({ from, to })}>{t('analytics.apply')}</Button>
      </div>

      <InlineNotice message={error} className="mb-5" />
      {loading && !overview ? (
        <PagePendingState variant="admin" label={t('common.loading')} />
      ) : overview ? (
        <div className="space-y-8">
          <div className="grid gap-3 sm:grid-cols-3">
            <Metric label={t('analytics.pv')} value={overview.summary.pv} change={overview.summary.pvChange} t={t} />
            <Metric label={t('analytics.uv')} value={overview.summary.uv} change={overview.summary.uvChange} t={t} />
            <Metric label={t('analytics.likes')} value={overview.summary.likes} change={overview.summary.likesChange} t={t} />
          </div>

          <TrendChart data={overview.trend} />

          <div className="grid gap-6 xl:grid-cols-2">
            <Panel title={t('analytics.topPages')}>
              {overview.topPages.map((item) => (
                <div key={`${item.path}-${item.articleId || 0}`} className="flex justify-between gap-4 border-b border-hairline py-3 text-sm">
                  <span className="truncate text-ink">{item.title || item.path}</span>
                  <span className="shrink-0 text-muted">{item.pv} PV · {item.uv} UV</span>
                </div>
              ))}
            </Panel>
            <Panel title={t('analytics.referers')}>
              {overview.topReferers.map((item) => (
                <div key={`${item.sourceType}-${item.sourceName}`} className="flex justify-between gap-4 border-b border-hairline py-3 text-sm">
                  <span className="truncate text-ink">{item.sourceName}</span>
                  <span className="shrink-0 text-muted">{item.pv} PV</span>
                </div>
              ))}
            </Panel>
          </div>

          <Panel title={t('analytics.topArticles')}>
            <div className="overflow-x-auto">
              <table className="w-full text-left text-sm">
                <thead>
                  <tr className="border-b border-hairline text-muted">
                    <th className="py-3">{t('adminArticles.title')}</th>
                    <th>PV</th>
                    <th>UV</th>
                    <th>{t('analytics.likes')}</th>
                  </tr>
                </thead>
                <tbody>
                  {articles.map((article) => (
                    <tr key={article.articleId} className="border-b border-hairline hover:bg-surface-soft">
                      <td className="py-3">
                        <button type="button" className="text-left text-ink hover:text-ochre" onClick={() => void openDetail(article)}>
                          {article.title}
                        </button>
                      </td>
                      <td>{article.pv}</td>
                      <td>{article.uv}</td>
                      <td>{article.likes}</td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          </Panel>

          {detail && (
            <Panel title={detail.article.title}>
              <TrendChart data={detail.trend} />
              <h5 className="mt-5 font-medium text-ink">{t('analytics.referers')}</h5>
              <div className="mt-2">
                {detail.referers.map((item) => (
                  <div key={`${item.sourceType}-${item.sourceName}`} className="flex justify-between border-b border-hairline py-2 text-sm">
                    <span>{item.sourceName}</span>
                    <span className="text-muted">{item.pv} PV</span>
                  </div>
                ))}
              </div>
              <div className="mt-4 flex justify-end">
                <Button size="sm" variant="subtle" onClick={() => setDetail(null)}>{t('common.dismiss')}</Button>
              </div>
            </Panel>
          )}
        </div>
      ) : null}
    </div>
  );
};

const Metric = ({ label, value, change, t }: {
  label: string;
  value: number;
  change?: number;
  t: (key: Parameters<typeof translate>[1]) => string;
}) => (
  <div className="rounded-lg bg-paper p-5 shadow-xs">
    <p className="text-xs text-muted">{label}</p>
    <p className="mt-2 font-display text-4xl text-ink">{value}</p>
    <p className="mt-2 text-xs text-muted">
      {typeof change === 'number' ? formatText(t('analytics.change'), { value: change }) : t('analytics.noComparison')}
    </p>
  </div>
);

const Panel = ({ title, children }: { title: string; children: React.ReactNode }) => (
  <section className="rounded-lg bg-paper p-5 shadow-xs">
    <h4 className="mb-3 font-display text-2xl text-ink">{title}</h4>
    {children}
  </section>
);

const TrendChart = ({ data }: { data: Array<{ date: string; pv: number; uv: number }> }) => {
  const containerRef = useRef<HTMLDivElement>(null);
  const chartRef = useRef<ECharts | null>(null);

  useEffect(() => {
    if (!containerRef.current || data.length === 0) {
      chartRef.current?.clear();
      return;
    }
    let active = true;
    let resize: (() => void) | undefined;
    void loadECharts().then(({ init }) => {
      if (!active || !containerRef.current) {
        return;
      }
      chartRef.current ??= init(containerRef.current);
      chartRef.current.setOption({
        tooltip: { trigger: 'axis' },
        legend: { data: ['PV', 'UV'] },
        grid: { left: 44, right: 18, top: 40, bottom: 32 },
        xAxis: { type: 'category', data: data.map((point) => point.date) },
        yAxis: { type: 'value', minInterval: 1 },
        series: [
          { name: 'PV', type: 'line', smooth: true, data: data.map((point) => point.pv) },
          { name: 'UV', type: 'line', smooth: true, data: data.map((point) => point.uv) },
        ],
      });
      resize = () => chartRef.current?.resize();
      window.addEventListener('resize', resize);
    }).catch(() => undefined);
    return () => {
      active = false;
      if (resize) {
        window.removeEventListener('resize', resize);
      }
    };
  }, [data]);

  useEffect(() => () => {
    chartRef.current?.dispose();
    chartRef.current = null;
  }, []);

  return <div ref={containerRef} className="h-72 w-full" />;
};

export default AdminAnalytics;
