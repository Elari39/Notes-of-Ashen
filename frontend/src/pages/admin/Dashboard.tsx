import React, { useEffect, useRef, useState } from 'react';
import { Link } from 'react-router-dom';
import { LineChart } from 'echarts/charts';
import { GridComponent, LegendComponent, TooltipComponent } from 'echarts/components';
import { init, use } from 'echarts/core';
import { CanvasRenderer } from 'echarts/renderers';
import type { ECharts } from 'echarts/core';
import { getAdminStats } from '../../api/admin';
import InlineNotice from '../../components/InlineNotice';
import PagePendingState from '../../components/RoutePending';
import Tag from '../../components/ui/Tag';
import { getArticleStatusLabel, getDateLocale, translate } from '../../i18n';
import { usePreferenceStore, type Language } from '../../store/preferences';
import { getErrorMessage } from '../../utils/error';
import type { AdminStats, Article, Log, RefererStat, TrafficTrendPoint } from '../../types';
import { formatLogResource, getLogEventPresentation } from './logPresentation';

use([LineChart, GridComponent, TooltipComponent, LegendComponent, CanvasRenderer]);

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
      <div className="mb-8 border-b border-hairline pb-5">
        <p className="editorial-kicker mb-3">{t('dashboard.kicker')}</p>
        <h3 className="text-4xl text-ink">{t('dashboard.title')}</h3>
      </div>

      <InlineNotice message={error} className="mb-6" />
      {loading && <PagePendingState variant="inline" label={t('dashboard.loading')} />}

      {stats && (
        <div className="space-y-10">
          <div className="grid grid-cols-2 gap-3 md:grid-cols-3 xl:grid-cols-5">
            {statItems.map((key) => (
              <div key={key} className="rounded-lg bg-paper p-4 shadow-xs">
                <p className="text-xs font-medium text-muted">{t(`dashboard.${key}`)}</p>
                <p className="mt-3 font-display text-4xl text-ink">{stats[key]}</p>
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

const TrafficOverview: React.FC<{
  trend: TrafficTrendPoint[];
  referers: RefererStat[];
  todayPv: number;
  todayUv: number;
  language: Language;
}> = ({ trend, referers, todayPv, todayUv, language }) => {
  const t = (key: Parameters<typeof translate>[1]) => translate(language, key);
  const sourceTotal = referers.reduce((sum, item) => sum + item.pv, 0);
  const chartRef = useRef<HTMLDivElement>(null);
  const chartInstance = useRef<ECharts | null>(null);
  const resizeHandler = useRef<(() => void) | null>(null);

  // 数据 effect：确保实例存在并应用 option。每次 trend/language 变化或挂载后都重跑，
  // 避免原「init effect 依赖 [] + setOption effect 依赖 [trend,language]」在 StrictMode
  // 重挂载后新实例拿不到 option 导致图表空白。实例为空时懒初始化并注册 resize 监听。
  useEffect(() => {
    if (!chartRef.current) {
      return;
    }
    let chart = chartInstance.current;
    if (!chart) {
      chart = init(chartRef.current, undefined, { renderer: 'canvas' });
      chartInstance.current = chart;
      const resize = () => chart?.resize();
      resizeHandler.current = resize;
      window.addEventListener('resize', resize);
    }
    chart.setOption({
      tooltip: {
        trigger: 'axis',
        backgroundColor: 'var(--paper)',
        borderColor: 'var(--mountain-grey)',
        borderWidth: 1,
        textStyle: { color: 'var(--ink)' },
        formatter: (params: { axisValue: string; marker: string; seriesName: string; value: number }[]) => {
          const title = fullDate(String(params[0]?.axisValue ?? ''), language);
          const lines = params
            .map((p) => `${p.marker} ${p.seriesName}: ${p.value}`)
            .join('<br/>');
          return `${title}<br/>${lines}`;
        },
      },
      legend: {
        data: ['PV', 'UV'],
        textStyle: { color: 'var(--ink-light)', fontSize: 11 },
        top: 0,
      },
      grid: { top: 32, right: 16, bottom: 8, left: 36, containLabel: true },
      xAxis: {
        type: 'category',
        boundaryGap: false,
        data: trend.map((p) => p.date),
        axisLabel: {
          color: 'var(--ink-light)',
          fontSize: 11,
          formatter: (value: string) => shortDate(value, language),
        },
        axisLine: { lineStyle: { color: 'var(--mountain-grey)' } },
      },
      yAxis: {
        type: 'value',
        minInterval: 1,
        axisLabel: { color: 'var(--ink-light)', fontSize: 11 },
        splitLine: { lineStyle: { color: 'var(--mountain-grey)', type: 'dashed' } },
      },
      series: [
        {
          name: 'PV',
          type: 'line',
          smooth: true,
          showSymbol: false,
          data: trend.map((p) => p.pv),
          lineStyle: { width: 2, color: 'var(--ochre)' },
          itemStyle: { color: 'var(--ochre)' },
        },
        {
          name: 'UV',
          type: 'line',
          smooth: true,
          showSymbol: false,
          data: trend.map((p) => p.uv),
          lineStyle: { width: 2, color: 'var(--code-blue)' },
          itemStyle: { color: 'var(--code-blue)' },
        },
      ],
    }, true);
  }, [trend, language]);

  // 卸载清理：dispose 实例并移除 resize 监听。实例在数据 effect 中懒创建，
  // 故通过 ref 在卸载时取最新 handler，仅依赖 [] 保证组件卸载时执行一次。
  useEffect(() => {
    return () => {
      if (resizeHandler.current) {
        window.removeEventListener('resize', resizeHandler.current);
        resizeHandler.current = null;
      }
      chartInstance.current?.dispose();
      chartInstance.current = null;
    };
  }, []);

  return (
    <section>
      <div className="mb-4 flex flex-col gap-3 sm:flex-row sm:items-end sm:justify-between">
        <div>
          <h4 className="text-sm font-bold tracking-widest text-ink">{t('dashboard.trafficTitle')}</h4>
          <p className="mt-2 text-xs text-ink-light">{t('dashboard.trafficSubtitle')}</p>
        </div>
        <div className="grid grid-cols-2 gap-2 text-sm">
          <div className="rounded-md bg-paper px-4 py-3">
            <span className="block text-xs text-ink-light">{t('dashboard.todayPv')}</span>
            <strong className="mt-1 block text-xl text-ink">{todayPv}</strong>
          </div>
          <div className="rounded-md bg-paper px-4 py-3">
            <span className="block text-xs text-ink-light">{t('dashboard.todayUv')}</span>
            <strong className="mt-1 block text-xl text-ink">{todayUv}</strong>
          </div>
        </div>
      </div>

      <div className="grid gap-6 lg:grid-cols-[minmax(0,2fr)_minmax(16rem,1fr)]">
        <div ref={chartRef} className="h-72 rounded-lg bg-paper p-3 shadow-xs" />

        <div className="rounded-lg bg-paper p-5 shadow-xs">
          <h5 className="mb-4 text-xs font-bold tracking-widest text-ink">{t('dashboard.referers')}</h5>
          {referers.length === 0 ? (
            <p className="text-sm text-ink-light">{t('dashboard.empty')}</p>
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

const LogRow: React.FC<{ log: Log; language: Language }> = ({ log, language }) => {
  const presentation = getLogEventPresentation(log.eventType, language);
  return (
    <div className="grid gap-2 border-b border-mountain-grey p-4 text-sm last:border-b-0 md:grid-cols-[1fr_auto] md:items-center">
      <div className="min-w-0">
        <div className="flex flex-wrap items-center gap-2">
          <Tag tone={presentation.tone} size="sm">{presentation.label}</Tag>
          <span className="truncate font-mono text-[0.7rem] text-ink-light opacity-70">{log.eventType}</span>
        </div>
        <p className="mt-2 text-xs text-ink-light">
          {log.userAccount || (log.userId ? `#${log.userId}` : translate(language, 'logs.anonymous'))}
          {' · '}{formatLogResource(log, language)}
          {log.ip ? ` · ${log.ip}` : ''}
        </p>
      </div>
      <time className="text-xs text-ink-light">{new Date(log.createdAt).toLocaleString(getDateLocale(language))}</time>
    </div>
  );
};

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
  if (item.sourceType === 'direct') return translate(language, 'dashboard.refererDirect');
  if (item.sourceType === 'internal') return translate(language, 'dashboard.refererInternal');
  if (item.sourceType === 'search') return `${translate(language, 'dashboard.refererSearch')}: ${item.sourceName}`;
  if (item.sourceType === 'external') return `${translate(language, 'dashboard.refererExternal')}: ${item.sourceName}`;
  return item.sourceName;
};

export default AdminDashboard;
