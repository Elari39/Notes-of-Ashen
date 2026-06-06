import React, { useEffect, useState } from 'react';
import { getUsers, updateUserStatus } from '../../api/user';
import { User } from '../../types';
import Pagination from '../../components/Pagination';
import InlineNotice from '../../components/InlineNotice';
import { getErrorMessage } from '../../utils/error';

const AdminUsers: React.FC = () => {
  const [users, setUsers] = useState<User[]>([]);
  const [page, setPage] = useState(1);
  const [total, setTotal] = useState(0);
  const [error, setError] = useState('');
  const [busyId, setBusyId] = useState<number | null>(null);
  const size = 10;

  const fetchList = async () => {
    try {
      const res = await getUsers({ page, size });
      setUsers(res.data.items || []);
      setTotal(res.data.total || 0);
    } catch (e) {
      setError(getErrorMessage(e, '用户列表加载失败'));
    }
  };

  useEffect(() => {
    fetchList();
  }, [page]);

  const handleStatus = async (id: number, currentStatus: string) => {
    const newStatus = currentStatus === 'active' ? 'disabled' : 'active';
    if (confirm(`确认${newStatus === 'active' ? '启用' : '禁用'}此人？`)) {
      setError('');
      setBusyId(id);
      try {
        await updateUserStatus(id, newStatus);
        fetchList();
      } catch (e: any) {
        setError(getErrorMessage(e, '操作失败'));
      } finally {
        setBusyId(null);
      }
    }
  };

  return (
    <div>
      <div className="mb-8 border-b border-mountain-grey pb-4">
        <h3 className="text-2xl font-bold text-ink tracking-widest">掌印 (用户)</h3>
      </div>

      <InlineNotice message={error} className="mb-6" />

      <table className="w-full text-left border-collapse text-sm">
        <thead>
          <tr className="border-b border-mountain-grey text-ink-light opacity-80 tracking-widest">
            <th className="py-3 font-normal">账文</th>
            <th className="py-3 font-normal">别号</th>
            <th className="py-3 font-normal">权柄</th>
            <th className="py-3 font-normal">状态</th>
            <th className="py-3 font-normal text-right">操作</th>
          </tr>
        </thead>
        <tbody>
          {users.map(u => (
            <tr key={u.id} className="border-b border-mountain-grey border-opacity-50 hover:bg-mountain-grey hover:bg-opacity-20 transition-colors text-ink">
              <td className="py-4 font-bold">{u.account}</td>
              <td className="py-4 text-ink-light">{u.nickname || '-'}</td>
              <td className="py-4 text-ink-light opacity-80">{u.role === 'admin' ? '掌卷' : '墨客'}</td>
              <td className="py-4">
                <span className={`px-2 py-1 text-xs border ${u.status === 'active' ? 'border-ochre text-ochre' : 'border-ink-light text-ink-light'}`}>
                  {u.status === 'active' ? '活跃' : '流放'}
                </span>
              </td>
              <td className="py-4 text-right space-x-4">
                {u.role !== 'admin' && (
                  <button onClick={() => handleStatus(u.id, u.status)} disabled={busyId === u.id} className="text-ochre opacity-80 hover:opacity-100 tracking-wider disabled:opacity-50 disabled:cursor-not-allowed">
                    {u.status === 'active' ? '流放' : '召回'}
                  </button>
                )}
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

export default AdminUsers;
