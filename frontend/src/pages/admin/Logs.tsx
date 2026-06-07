import React, { useCallback, useEffect, useState } from 'react';
import { getLogs } from '../../api/user';
import { Log } from '../../types';
import Pagination from '../../components/Pagination';
import InlineNotice from '../../components/InlineNotice';
import { getErrorMessage } from '../../utils/error';
import { getDateLocale, translate } from '../../i18n';
import { usePreferenceStore } from '../../store/preferences';

const AdminLogs: React.FC = () => {
  const language = usePreferenceStore((state) => state.language);
  const [logs, setLogs] = useState<Log[]>([]);
  const [page, setPage] = useState(1);
  const [total, setTotal] = useState(0);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');
  const size = 10;
  const t = (key: Parameters<typeof translate>[1]) => translate(language, key);

  const fetchList = useCallback(async () => {
    setLoading(true);
    setError('');
    try {
      const res = await getLogs({ page, size });
      setLogs(res.data.items || []);
      setTotal(res.data.total || 0);
    } catch (e) {
      setError(getErrorMessage(e, translate(language, 'logs.loadError')));
    } finally {
      setLoading(false);
    }
  }, [page, size, language]);

  useEffect(() => {
    fetchList();
  }, [fetchList]);

  return (
    <div>
      <div className="mb-8 border-b border-mountain-grey pb-4">
        <h3 className="text-2xl font-bold text-ink tracking-widest">{t('admin.logs')}</h3>
      </div>

      <InlineNotice message={error} className="mb-6" />

      {loading ? (
        <div className="py-16 text-center tracking-widest text-ink-light">{t('common.loading')}</div>
      ) : logs.length === 0 ? (
        <div className="py-16 text-center tracking-widest text-ink-light">{t('common.empty')}</div>
      ) : (
        <>
          <table className="admin-responsive-table w-full text-left border-collapse text-sm">
            <thead>
              <tr className="border-b border-mountain-grey text-ink-light opacity-80 tracking-widest">
                <th className="py-3 font-normal">{t('common.time')}</th>
                <th className="py-3 font-normal">{t('logs.user')}</th>
                <th className="py-3 font-normal">{t('logs.event')}</th>
                <th className="py-3 font-normal">{t('logs.source')}</th>
              </tr>
            </thead>
            <tbody>
              {logs.map(log => (
                <tr key={log.id} className="border-b border-mountain-grey border-opacity-50 hover:bg-mountain-grey hover:bg-opacity-20 transition-colors text-ink">
                  <td data-label={t('common.time')} className="admin-card-title py-4 text-ink-light whitespace-nowrap">{new Date(log.createdAt).toLocaleString(getDateLocale(language))}</td>
                  <td data-label={t('logs.user')} className="py-4 font-bold">{log.userId}</td>
                  <td data-label={t('logs.event')} className="py-4 text-ochre">{log.eventType}</td>
                  <td data-label={t('logs.source')} className="py-4 text-ink-light opacity-80 truncate max-w-[200px]" title={log.ip}>{log.ip}</td>
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
        </>
      )}
    </div>
  );
};

export default AdminLogs;
