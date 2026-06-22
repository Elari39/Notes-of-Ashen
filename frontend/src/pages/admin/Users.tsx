import React, { useCallback, useEffect, useRef, useState } from 'react';
import { getUsers, updateUserRole, updateUserStatus } from '../../api/user';
import { User } from '../../types';
import Pagination from '../../components/Pagination';
import InlineNotice from '../../components/InlineNotice';
import PagePendingState from '../../components/RoutePending';
import { getErrorMessage } from '../../utils/error';
import { formatText, getUserRoleLabel, getUserStatusLabel, translate } from '../../i18n';
import { usePreferenceStore } from '../../store/preferences';
import { useConfirm } from '../../hooks/useConfirm';

const AdminUsers: React.FC = () => {
  const language = usePreferenceStore((state) => state.language);
  const confirm = useConfirm();
  const [users, setUsers] = useState<User[]>([]);
  const [page, setPage] = useState(1);
  const [total, setTotal] = useState(0);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');
  const [busyId, setBusyId] = useState<number | null>(null);
  const listRequestRef = useRef(0);
  const size = 10;
  const t = (key: Parameters<typeof translate>[1]) => translate(language, key);

  const fetchList = useCallback(async () => {
    const requestId = listRequestRef.current + 1;
    listRequestRef.current = requestId;
    setLoading(true);
    setError('');
    try {
      const res = await getUsers({ page, size });
      if (listRequestRef.current !== requestId) {
        return;
      }
      setUsers(res.data.items || []);
      setTotal(res.data.total || 0);
    } catch (e) {
      if (listRequestRef.current === requestId) {
        setError(getErrorMessage(e, translate(language, 'users.loadError')));
      }
    } finally {
      if (listRequestRef.current === requestId) {
        setLoading(false);
      }
    }
  }, [page, size, language]);

  useEffect(() => {
    fetchList();
  }, [fetchList]);

  const handleStatus = async (id: number, currentStatus: string) => {
    const newStatus = currentStatus === 'active' ? 'disabled' : 'active';
    const action = newStatus === 'active' ? t('users.activate') : t('users.disable');
    const ok = await confirm({
      title: formatText(t('users.confirmStatus'), { action }),
      confirmLabel: t('common.confirm'),
      cancelLabel: t('common.cancel'),
      tone: 'danger',
    });
    if (!ok) return;
    setError('');
    setBusyId(id);
    try {
      await updateUserStatus(id, newStatus);
      await fetchList();
    } catch (e: unknown) {
      setError(getErrorMessage(e, t('users.actionError')));
    } finally {
      setBusyId(null);
    }
  };

  const handleRole = async (id: number, role: string) => {
    setError('');
    setBusyId(id);
    try {
      await updateUserRole(id, role);
      await fetchList();
    } catch (e: unknown) {
      setError(getErrorMessage(e, t('users.actionError')));
    } finally {
      setBusyId(null);
    }
  };

  return (
    <div>
      <div className="mb-8 border-b border-mountain-grey pb-4">
        <h3 className="text-2xl font-bold text-ink tracking-widest">{t('admin.users')}</h3>
      </div>

      <InlineNotice message={error} className="mb-6" />

      {loading && (
        <PagePendingState
          variant={users.length > 0 ? 'inline' : 'admin'}
          label={t('common.loading')}
        />
      )}
      {!loading && users.length === 0 ? (
        <div className="py-16 text-center tracking-widest text-ink-light">{t('common.empty')}</div>
      ) : users.length > 0 ? (
        <>
          <table className="admin-responsive-table w-full text-left border-collapse text-sm">
            <thead>
              <tr className="border-b border-mountain-grey text-ink-light opacity-80 tracking-widest">
                <th className="py-3 font-normal">{t('users.account')}</th>
                <th className="py-3 font-normal">{t('users.nickname')}</th>
                <th className="py-3 font-normal">{t('users.role')}</th>
                <th className="py-3 font-normal">{t('common.status')}</th>
                <th className="py-3 font-normal text-right">{t('common.action')}</th>
              </tr>
            </thead>
            <tbody>
              {users.map(user => (
                <tr key={user.id} className="border-b border-mountain-grey border-opacity-50 hover:bg-mountain-grey hover:bg-opacity-20 transition-colors text-ink">
                  <td data-label={t('users.account')} className="admin-card-title py-4 font-bold">{user.account}</td>
                  <td data-label={t('users.nickname')} className="py-4 text-ink-light">{user.nickname || '-'}</td>
                  <td data-label={t('users.role')} className="py-4 text-ink-light opacity-80">
                    <select
                      value={user.role}
                      disabled={busyId === user.id}
                      onChange={(event) => handleRole(user.id, event.target.value)}
                      className="bg-paper border border-mountain-grey px-2 py-1 text-sm text-ink outline-none focus:border-ochre disabled:opacity-50"
                    >
                      <option value="user">{getUserRoleLabel(language, 'user')}</option>
                      <option value="editor">{language === 'zh' ? '编辑' : 'Editor'}</option>
                      <option value="admin">{getUserRoleLabel(language, 'admin')}</option>
                    </select>
                  </td>
                  <td data-label={t('common.status')} className="py-4">
                    <span className={`px-2 py-1 text-xs border ${user.status === 'active' ? 'border-ochre text-ochre' : 'border-ink-light text-ink-light'}`}>
                      {getUserStatusLabel(language, user.status)}
                    </span>
                  </td>
                  <td data-label={t('common.action')} className="admin-card-actions py-4 text-right">
                    <div className="admin-action-list">
                      <button onClick={() => handleStatus(user.id, user.status)} disabled={busyId === user.id} className="text-ochre opacity-80 hover:opacity-100 tracking-wider disabled:opacity-50 disabled:cursor-not-allowed">
                        {user.status === 'active' ? t('users.disable') : t('users.activate')}
                      </button>
                    </div>
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
        </>
      ) : null}
    </div>
  );
};

export default AdminUsers;
