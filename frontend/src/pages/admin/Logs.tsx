import React, { useCallback, useEffect, useState } from 'react';
import { getLogs } from '../../api/user';
import { Log } from '../../types';
import Pagination from '../../components/Pagination';
import InlineNotice from '../../components/InlineNotice';
import { getErrorMessage } from '../../utils/error';

const AdminLogs: React.FC = () => {
  const [logs, setLogs] = useState<Log[]>([]);
  const [page, setPage] = useState(1);
  const [total, setTotal] = useState(0);
  const [error, setError] = useState('');
  const size = 10;

  const fetchList = useCallback(async () => {
    try {
      const res = await getLogs({ page, size });
      setLogs(res.data.items || []);
      setTotal(res.data.total || 0);
    } catch (e) {
      setError(getErrorMessage(e, '日志列表加载失败'));
    }
  }, [page, size]);

  useEffect(() => {
    fetchList();
  }, [fetchList]);

  return (
    <div>
      <div className="mb-8 border-b border-mountain-grey pb-4">
        <h3 className="text-2xl font-bold text-ink tracking-widest">青史 (日志)</h3>
      </div>

      <InlineNotice message={error} className="mb-6" />

      <table className="w-full text-left border-collapse text-sm">
        <thead>
          <tr className="border-b border-mountain-grey text-ink-light opacity-80 tracking-widest">
            <th className="py-3 font-normal">时间</th>
            <th className="py-3 font-normal">谁人</th>
            <th className="py-3 font-normal">何事</th>
            <th className="py-3 font-normal">溯源</th>
          </tr>
        </thead>
        <tbody>
          {logs.map(l => (
            <tr key={l.id} className="border-b border-mountain-grey border-opacity-50 hover:bg-mountain-grey hover:bg-opacity-20 transition-colors text-ink">
              <td className="py-4 text-ink-light whitespace-nowrap">{new Date(l.createdAt).toLocaleString('zh-CN')}</td>
              <td className="py-4 font-bold">{l.userId}</td>
              <td className="py-4 text-ochre">{l.eventType}</td>
              <td className="py-4 text-ink-light opacity-80 truncate max-w-[200px]" title={l.ip}>{l.ip}</td>
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

export default AdminLogs;
