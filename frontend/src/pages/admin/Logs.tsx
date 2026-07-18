import React, { useCallback, useEffect, useRef, useState } from 'react';
import { getLogs } from '../../api/user';
import type { Log } from '../../types';
import type { OperationLogListParams } from '../../types/api';
import Pagination from '../../components/Pagination';
import InlineNotice from '../../components/InlineNotice';
import PagePendingState from '../../components/RoutePending';
import TableSkeleton from '../../components/ui/TableSkeleton';
import EmptyState from '../../components/ui/EmptyState';
import Tag from '../../components/ui/Tag';
import { getErrorMessage } from '../../utils/error';
import { formatText, getDateLocale, translate } from '../../i18n';
import { usePreferenceStore } from '../../store/preferences';
import {
  dateToUTCBoundary,
  formatLogMetadataValue,
  formatLogResource,
  getClientSummary,
  getLogEventPresentation,
  getLogMetadataLabel,
  LOG_EVENT_TYPES,
  parseLogMetadata,
} from './logPresentation';

type LogFilterForm = {
  eventType: string;
  actor: string;
  ip: string;
  startDate: string;
  endDate: string;
};

const emptyFilterForm: LogFilterForm = {
  eventType: '',
  actor: '',
  ip: '',
  startDate: '',
  endDate: '',
};

const AdminLogs: React.FC = () => {
  const language = usePreferenceStore((state) => state.language);
  const [logs, setLogs] = useState<Log[]>([]);
  const [page, setPage] = useState(1);
  const [total, setTotal] = useState(0);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');
  const [filterForm, setFilterForm] = useState<LogFilterForm>(emptyFilterForm);
  const [appliedFilters, setAppliedFilters] = useState<OperationLogListParams>({});
  const [expandedLogId, setExpandedLogId] = useState<number | null>(null);
  const listRequestRef = useRef(0);
  const size = 10;
  const t = (key: Parameters<typeof translate>[1]) => translate(language, key);
  const hasAppliedFilters = Object.keys(appliedFilters).length > 0;

  const fetchList = useCallback(async () => {
    const requestId = listRequestRef.current + 1;
    listRequestRef.current = requestId;
    setLoading(true);
    setError('');
    try {
      const res = await getLogs({ page, size, ...appliedFilters });
      if (listRequestRef.current !== requestId) {
        return;
      }
      setLogs(res.data.items || []);
      setTotal(res.data.total || 0);
    } catch (e) {
      if (listRequestRef.current === requestId) {
        setError(getErrorMessage(e, translate(language, 'logs.loadError')));
      }
    } finally {
      if (listRequestRef.current === requestId) {
        setLoading(false);
      }
    }
  }, [page, size, appliedFilters, language]);

  useEffect(() => {
    fetchList();
  }, [fetchList]);

  const updateFilter = (key: keyof LogFilterForm, value: string) => {
    setFilterForm((current) => ({ ...current, [key]: value }));
  };

  const handleFilterSubmit = (event: React.FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    if (filterForm.startDate && filterForm.endDate && filterForm.startDate > filterForm.endDate) {
      setError(t('logs.invalidDateRange'));
      return;
    }
    const startAt = dateToUTCBoundary(filterForm.startDate, false);
    const endAt = dateToUTCBoundary(filterForm.endDate, true);
    setError('');
    setAppliedFilters({
      ...(filterForm.eventType ? { eventType: filterForm.eventType } : {}),
      ...(filterForm.actor.trim() ? { actor: filterForm.actor.trim() } : {}),
      ...(filterForm.ip.trim() ? { ip: filterForm.ip.trim() } : {}),
      ...(startAt ? { startAt } : {}),
      ...(endAt ? { endAt } : {}),
    });
    setPage(1);
    setExpandedLogId(null);
  };

  const handleClearFilters = () => {
    setFilterForm(emptyFilterForm);
    setAppliedFilters({});
    setPage(1);
    setExpandedLogId(null);
    setError('');
  };

  const handlePageChange = (nextPage: number) => {
    setPage(nextPage);
    setExpandedLogId(null);
  };

  return (
    <div>
      <div className="mb-6 border-b border-mountain-grey pb-4">
        <div className="flex flex-col gap-2 sm:flex-row sm:items-end sm:justify-between">
          <div>
            <h3 className="text-2xl font-bold tracking-widest text-ink">{t('admin.logs')}</h3>
            <p className="mt-2 text-sm text-ink-light">{t('logs.subtitle')}</p>
          </div>
          <p className="text-xs tracking-widest text-ink-light">
            {formatText(t('logs.total'), { total })}
          </p>
        </div>
      </div>

      <form
        onSubmit={handleFilterSubmit}
        className="mb-6 grid grid-cols-1 gap-3 rounded-md border border-mountain-grey bg-surface-soft p-4 md:grid-cols-2 xl:grid-cols-6"
      >
        <label className="flex flex-col gap-1 text-xs tracking-wider text-ink-light">
          <span>{t('logs.filterEvent')}</span>
          <select
            value={filterForm.eventType}
            onChange={(event) => updateFilter('eventType', event.target.value)}
            className="min-h-10 bg-paper border border-mountain-grey px-3 py-2 text-sm text-ink outline-hidden focus:border-ochre"
          >
            <option value="">{t('logs.allEvents')}</option>
            {LOG_EVENT_TYPES.map((eventType) => (
              <option key={eventType} value={eventType}>
                {getLogEventPresentation(eventType, language).label}
              </option>
            ))}
          </select>
        </label>
        <label className="flex flex-col gap-1 text-xs tracking-wider text-ink-light">
          <span>{t('logs.filterActor')}</span>
          <input
            value={filterForm.actor}
            onChange={(event) => updateFilter('actor', event.target.value)}
            className="min-h-10 bg-paper border border-mountain-grey px-3 py-2 text-sm text-ink outline-hidden focus:border-ochre"
          />
        </label>
        <label className="flex flex-col gap-1 text-xs tracking-wider text-ink-light">
          <span>{t('logs.filterIp')}</span>
          <input
            value={filterForm.ip}
            onChange={(event) => updateFilter('ip', event.target.value)}
            inputMode="text"
            className="min-h-10 bg-paper border border-mountain-grey px-3 py-2 font-mono text-sm text-ink outline-hidden focus:border-ochre"
          />
        </label>
        <label className="flex flex-col gap-1 text-xs tracking-wider text-ink-light">
          <span>{t('logs.startDate')}</span>
          <input
            type="date"
            value={filterForm.startDate}
            onChange={(event) => updateFilter('startDate', event.target.value)}
            className="min-h-10 bg-paper border border-mountain-grey px-3 py-2 text-sm text-ink outline-hidden focus:border-ochre"
          />
        </label>
        <label className="flex flex-col gap-1 text-xs tracking-wider text-ink-light">
          <span>{t('logs.endDate')}</span>
          <input
            type="date"
            value={filterForm.endDate}
            onChange={(event) => updateFilter('endDate', event.target.value)}
            className="min-h-10 bg-paper border border-mountain-grey px-3 py-2 text-sm text-ink outline-hidden focus:border-ochre"
          />
        </label>
        <div className="flex items-end gap-2">
          <button
            type="submit"
            className="min-h-10 flex-1 border border-ink bg-ink px-4 py-2 text-sm tracking-widest text-paper transition-colors hover:bg-transparent hover:text-ink focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-ochre"
          >
            {t('logs.search')}
          </button>
          <button
            type="button"
            onClick={handleClearFilters}
            className="min-h-10 flex-1 border border-mountain-grey px-4 py-2 text-sm tracking-widest text-ink transition-colors hover:border-ochre hover:text-ochre focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-ochre"
          >
            {t('logs.reset')}
          </button>
        </div>
      </form>

      <InlineNotice message={error} className="mb-6" />

      {loading && logs.length === 0 && <TableSkeleton rows={5} cols={6} />}
      {loading && logs.length > 0 && (
        <PagePendingState variant="inline" label={t('common.loading')} />
      )}
      {!loading && logs.length === 0 ? (
        <EmptyState
          illustration="cloud"
          title={t(hasAppliedFilters ? 'logs.filteredEmptyTitle' : 'logs.emptyTitle')}
          description={t(hasAppliedFilters ? 'logs.filteredEmptyDescription' : 'logs.emptyDescription')}
          action={hasAppliedFilters ? { label: t('logs.clearFilters'), onClick: handleClearFilters } : undefined}
        />
      ) : logs.length > 0 ? (
        <>
          <table className="admin-responsive-table w-full text-left border-collapse text-sm">
            <thead>
              <tr className="border-b border-mountain-grey text-ink-light opacity-80 tracking-widest">
                <th className="py-3 font-normal">{t('logs.event')}</th>
                <th className="py-3 font-normal">{t('logs.user')}</th>
                <th className="py-3 font-normal">{t('logs.resource')}</th>
                <th className="py-3 font-normal">{t('logs.source')}</th>
                <th className="py-3 font-normal">{t('common.time')}</th>
                <th className="py-3 font-normal text-right">{t('logs.details')}</th>
              </tr>
            </thead>
            <tbody>
              {logs.map((log) => {
                const eventPresentation = getLogEventPresentation(log.eventType, language);
                const expanded = expandedLogId === log.id;
                const detailId = `operation-log-detail-${log.id}`;
                const metadata = parseLogMetadata(log.metadata);
                return (
                  <React.Fragment key={log.id}>
                    <tr className={`log-main-row border-b border-mountain-grey/50 hover:bg-mountain-grey/20 transition-colors text-ink ${expanded ? 'log-main-row-expanded' : ''}`}>
                      <td data-label={t('logs.event')} className="admin-card-title py-4">
                        <Tag tone={eventPresentation.tone} size="sm">{eventPresentation.label}</Tag>
                        <div className="mt-1 break-all font-mono text-[0.7rem] text-ink-light opacity-70">
                          {log.eventType || t('logs.unknownEvent')}
                        </div>
                      </td>
                      <td data-label={t('logs.user')} className="py-4 font-bold">
                        {log.userAccount || (log.userId ? `#${log.userId}` : t('logs.anonymous'))}
                      </td>
                      <td data-label={t('logs.resource')} className="py-4 text-ink-light">
                        {formatLogResource(log, language)}
                      </td>
                      <td data-label={t('logs.source')} className="max-w-[200px] py-4 font-mono text-xs text-ink-light">
                        <span className="block truncate" title={log.ip || t('logs.noIp')}>
                          {log.ip || t('logs.noIp')}
                        </span>
                      </td>
                      <td data-label={t('common.time')} className="py-4 text-ink-light whitespace-nowrap">
                        {new Date(log.createdAt).toLocaleString(getDateLocale(language))}
                      </td>
                      <td data-label={t('logs.details')} className="admin-card-actions py-4 text-right">
                        <button
                          type="button"
                          aria-expanded={expanded}
                          aria-controls={detailId}
                          onClick={() => setExpandedLogId(expanded ? null : log.id)}
                          className="inline-flex min-h-10 items-center gap-2 px-2 text-xs tracking-wider text-ink-light transition-colors hover:text-ochre focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-ochre"
                        >
                          {t(expanded ? 'logs.collapse' : 'logs.expand')}
                          <svg
                            viewBox="0 0 16 16"
                            width="14"
                            height="14"
                            aria-hidden="true"
                            className={`transition-transform ${expanded ? 'rotate-180' : ''}`}
                          >
                            <path d="m3 6 5 5 5-5" fill="none" stroke="currentColor" strokeWidth="1.5" />
                          </svg>
                        </button>
                      </td>
                    </tr>
                    {expanded && (
                      <tr className="log-detail-row border-b border-mountain-grey text-ink">
                        <td colSpan={6} className="bg-surface-soft px-4 py-5">
                          <div id={detailId} className="space-y-5">
                            <dl className="grid gap-4 sm:grid-cols-2 xl:grid-cols-4">
                              <LogDetail label={t('logs.logId')} value={`#${log.id}`} mono />
                              <LogDetail
                                label={t('logs.userId')}
                                value={log.userId ? `#${log.userId}` : t('logs.anonymous')}
                                mono
                              />
                              <LogDetail label={t('logs.resource')} value={formatLogResource(log, language)} />
                              <LogDetail
                                label={t('common.time')}
                                value={new Date(log.createdAt).toLocaleString(getDateLocale(language))}
                              />
                              <LogDetail label={t('logs.source')} value={log.ip || t('logs.noIp')} mono />
                              <LogDetail label={t('logs.client')} value={getClientSummary(log.userAgent, language)} />
                            </dl>

                            <div>
                              <h4 className="mb-2 text-xs font-bold tracking-widest text-ink-light">{t('logs.metadata')}</h4>
                              {metadata.entries.length > 0 ? (
                                <dl className="grid gap-2 rounded-md border border-mountain-grey bg-paper p-3 sm:grid-cols-2">
                                  {metadata.entries.map((entry) => (
                                    <div key={entry.key} className="grid grid-cols-[minmax(5rem,auto)_1fr] gap-3 text-sm">
                                      <dt className="text-ink-light">{getLogMetadataLabel(entry.key, language)}</dt>
                                      <dd className="break-all font-mono text-ink">
                                        {formatLogMetadataValue(entry.key, entry.value, language)}
                                      </dd>
                                    </div>
                                  ))}
                                </dl>
                              ) : metadata.invalid ? (
                                <div className="rounded-md border border-mountain-grey bg-paper p-3">
                                  <p className="mb-2 text-xs text-ink-light">{t('logs.rawMetadata')}</p>
                                  <pre className="whitespace-pre-wrap break-all font-mono text-xs text-ink">{metadata.raw}</pre>
                                </div>
                              ) : (
                                <p className="text-sm text-ink-light">{t('logs.noMetadata')}</p>
                              )}
                            </div>

                            <div>
                              <h4 className="mb-2 text-xs font-bold tracking-widest text-ink-light">{t('logs.userAgent')}</h4>
                              <p className="break-all rounded-md border border-mountain-grey bg-paper p-3 font-mono text-xs leading-relaxed text-ink-light">
                                {log.userAgent || t('logs.unknownClient')}
                              </p>
                            </div>
                          </div>
                        </td>
                      </tr>
                    )}
                  </React.Fragment>
                );
              })}
            </tbody>
          </table>
          <Pagination
            currentPage={page}
            total={total}
            pageSize={size}
            onPageChange={handlePageChange}
          />
        </>
      ) : null}
    </div>
  );
};

const LogDetail: React.FC<{ label: string; value: string; mono?: boolean }> = ({ label, value, mono }) => (
  <div>
    <dt className="mb-1 text-xs tracking-wider text-ink-light">{label}</dt>
    <dd className={`break-all text-sm text-ink ${mono ? 'font-mono' : ''}`}>{value}</dd>
  </div>
);

export default AdminLogs;
