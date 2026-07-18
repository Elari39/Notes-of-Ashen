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
import { getChartThemeColors } from '../../utils/chartTheme';
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
                  <span className="shrink-0 text-muted">{item.pv} {t('analytics.pv')} · {item.uv} {t('analytics.uv')}</span>
                </div>
              ))}
            </Panel>
            <Panel title={t('analytics.referers')}>
              {overview.topReferers.map((item) => (
                <div key={`${item.sourceType}-${item.sourceName}`} className="flex justify-between gap-4 border-b border-hairline py-3 text-sm">
                  <span className="truncate text-ink">{item.sourceName}</span>
                  <span className="shrink-0 text-muted">{item.pv} {t('analytics.pv')}</span>
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
                    <th>{t('analytics.pv')}</th>
                    <th>{t('analytics.uv')}</th>
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
                    <span className="text-muted">{item.pv} {t('analytics.pv')}</span>
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
  const language = usePreferenceStore((state) => state.language);
  const effectiveTheme = usePreferenceStore((state) => state.effectiveTheme);
  const accentColor = usePreferenceStore((state) => state.accentColor);
  const t = (key: Parameters<typeof translate>[1]) => translate(language, key);
  const containerRef = useRef<HTMLDivElement>(null);
  const chartRef = useRef<ECharts | null>(null);
  const [highlightedSeries, setHighlightedSeries] = useState<'pv' | 'uv' | null>(null);
  const pvLabel = t('analytics.pv');
  const uvLabel = t('analytics.uv');

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
      const colors = getChartThemeColors();
      const seriesOption = (key: 'pv' | 'uv', label: string, values: number[], color: string) => {
        const isHighlighted = highlightedSeries === key;
        const isMuted = highlightedSeries !== null && !isHighlighted;
        return {
          name: label,
          type: 'line' as const,
          smooth: true,
          showSymbol: true,
          symbol: 'circle',
          symbolSize: isHighlighted ? 6 : 5,
          data: values,
          z: isHighlighted ? 3 : 2,
          lineStyle: { width: isHighlighted ? 3.5 : 2.5, color, opacity: isMuted ? 0.34 : 1 },
          itemStyle: {
            color,
            borderColor: colors.canvas,
            borderWidth: 1.5,
            opacity: isMuted ? 0.42 : 1,
          },
          emphasis: {
            focus: 'series' as const,
            lineStyle: { width: 4, color, opacity: 1 },
            itemStyle: { color, borderColor: colors.canvas, borderWidth: 2.5, opacity: 1 },
          },
        };
      };
      chartRef.current.setOption({
        animationDurationUpdate: 180,
        tooltip: {
          trigger: 'axis',
          confine: true,
          backgroundColor: colors.canvas,
          borderColor: colors.hairline,
          borderWidth: 1,
          padding: [8, 10],
          extraCssText: 'box-shadow: 0 8px 24px rgba(20, 20, 19, 0.14); border-radius: 8px;',
          textStyle: { color: colors.ink },
          axisPointer: {
            type: 'line',
            lineStyle: { color: colors.hairline, type: 'dashed', width: 1 },
            label: { backgroundColor: colors.ink, color: colors.canvas },
          },
        },
        grid: { left: 44, right: 18, top: 12, bottom: 32 },
        xAxis: {
          type: 'category',
          data: data.map((point) => point.date),
          axisLabel: { color: colors.muted, fontSize: 11 },
          axisLine: { lineStyle: { color: colors.hairline } },
          axisTick: { lineStyle: { color: colors.hairline } },
        },
        yAxis: {
          type: 'value',
          minInterval: 1,
          axisLabel: { color: colors.muted, fontSize: 11 },
          splitLine: { lineStyle: { color: colors.hairline, type: 'dashed' } },
        },
        series: [
          seriesOption('pv', pvLabel, data.map((point) => point.pv), colors.pv),
          seriesOption('uv', uvLabel, data.map((point) => point.uv), colors.uv),
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
  }, [data, effectiveTheme, accentColor, highlightedSeries, pvLabel, uvLabel]);

  useEffect(() => () => {
    chartRef.current?.dispose();
    chartRef.current = null;
  }, []);

  return (
    <div className="w-full">
      <div className="mb-3 flex flex-wrap items-center gap-2" role="group" aria-label={t('dashboard.trafficLegend')}>
        {([
          ['pv', pvLabel, 'bg-[var(--ochre)]'],
          ['uv', uvLabel, 'bg-[var(--accent-teal)]'],
        ] as const).map(([key, label, dotClass]) => {
          const isHighlighted = highlightedSeries === key;
          const isMuted = highlightedSeries !== null && !isHighlighted;
          return (
            <button
              key={key}
              type="button"
              aria-pressed={isHighlighted}
              title={t('dashboard.trafficLegendHelp')}
              onClick={() => setHighlightedSeries((current) => (current === key ? null : key))}
              style={isHighlighted ? { borderColor: key === 'pv' ? 'var(--ochre)' : 'var(--accent-teal)' } : undefined}
              className={`inline-flex min-h-8 items-center gap-2 rounded-md border px-2.5 py-1 text-xs transition-opacity ${
                isHighlighted ? 'bg-[var(--inline-code-bg)] text-ink' : 'border-hairline bg-paper text-ink-light'
              } ${isMuted ? 'opacity-40' : 'opacity-100'}`}
            >
              <span aria-hidden="true" className={`h-2.5 w-2.5 rounded-full ${dotClass}`} />
              {label}
            </button>
          );
        })}
      </div>
      <div ref={containerRef} role="img" className="h-64 w-full" aria-label={t('analytics.title')} />
    </div>
  );
};

export default AdminAnalytics;
