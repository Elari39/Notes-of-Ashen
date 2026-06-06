import React, { useCallback, useEffect, useState } from 'react';
import { getUsers, updateUserStatus } from '../../api/user';
import { User } from '../../types';
import Pagination from '../../components/Pagination';
import InlineNotice from '../../components/InlineNotice';
import { getErrorMessage } from '../../utils/error';
import { formatText, getUserRoleLabel, getUserStatusLabel, translate } from '../../i18n';
import { usePreferenceStore } from '../../store/preferences';
import { useSiteSettingsStore } from '../../store/siteSettings';

const AdminUsers: React.FC = () => {
  const language = usePreferenceStore((state) => state.language);
  const {
    registrationEnabled,
    isLoading: settingsLoading,
    fetchSettings,
    setRegistrationEnabled,
  } = useSiteSettingsStore();
  const [users, setUsers] = useState<User[]>([]);
  const [page, setPage] = useState(1);
  const [total, setTotal] = useState(0);
  const [error, setError] = useState('');
  const [settingsError, setSettingsError] = useState('');
  const [settingsNotice, setSettingsNotice] = useState('');
  const [busyId, setBusyId] = useState<number | null>(null);
  const size = 10;
  const t = (key: Parameters<typeof translate>[1]) => translate(language, key);

  const fetchList = useCallback(async () => {
    try {
      const res = await getUsers({ page, size });
      setUsers(res.data.items || []);
      setTotal(res.data.total || 0);
    } catch (e) {
      setError(getErrorMessage(e, translate(language, 'users.loadError')));
    }
  }, [page, size, language]);

  useEffect(() => {
    fetchList();
  }, [fetchList]);

  useEffect(() => {
    fetchSettings();
  }, [fetchSettings]);

  const handleRegistrationToggle = async () => {
    const nextEnabled = !registrationEnabled;
    setSettingsError('');
    setSettingsNotice('');
    try {
      await setRegistrationEnabled(nextEnabled);
      setSettingsNotice(nextEnabled ? t('users.registrationEnabledSaved') : t('users.registrationDisabledSaved'));
    } catch (e: unknown) {
      setSettingsError(getErrorMessage(e, t('users.registrationSettingsError')));
    }
  };

  const handleStatus = async (id: number, currentStatus: string) => {
    const newStatus = currentStatus === 'active' ? 'disabled' : 'active';
    const action = newStatus === 'active' ? t('users.activate') : t('users.disable');
    if (confirm(formatText(t('users.confirmStatus'), { action }))) {
      setError('');
      setBusyId(id);
      try {
        await updateUserStatus(id, newStatus);
        fetchList();
      } catch (e: unknown) {
        setError(getErrorMessage(e, t('users.actionError')));
      } finally {
        setBusyId(null);
      }
    }
  };

  return (
    <div>
      <div className="mb-8 border-b border-mountain-grey pb-4">
        <h3 className="text-2xl font-bold text-ink tracking-widest">{t('admin.users')}</h3>
      </div>

      <section className="mb-8 border border-mountain-grey bg-[var(--paper-soft)] p-5">
        <div className="flex flex-col gap-4 md:flex-row md:items-center md:justify-between">
          <div>
            <h4 className="text-base font-bold tracking-widest text-ink">{t('users.registrationSettingsTitle')}</h4>
            <p className="mt-2 text-sm leading-relaxed text-ink-light opacity-80">
              {t('users.registrationSettingsDesc')}
            </p>
          </div>
          <button
            type="button"
            role="switch"
            aria-checked={registrationEnabled}
            disabled={settingsLoading}
            onClick={handleRegistrationToggle}
            className={`min-w-32 border px-4 py-2 text-sm tracking-widest transition-colors disabled:cursor-not-allowed disabled:opacity-50 ${
              registrationEnabled
                ? 'border-ochre text-ochre hover:bg-ochre hover:text-paper'
                : 'border-ink-light text-ink-light hover:border-ink hover:text-ink'
            }`}
          >
            {settingsLoading ? t('common.saving') : registrationEnabled ? t('users.registrationEnabled') : t('users.registrationDisabled')}
          </button>
        </div>
        <InlineNotice message={settingsError} className="mt-4" />
        <InlineNotice message={settingsNotice} tone="success" className="mt-4" />
      </section>

      <InlineNotice message={error} className="mb-6" />

      <table className="w-full text-left border-collapse text-sm">
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
              <td className="py-4 font-bold">{user.account}</td>
              <td className="py-4 text-ink-light">{user.nickname || '-'}</td>
              <td className="py-4 text-ink-light opacity-80">{getUserRoleLabel(language, user.role)}</td>
              <td className="py-4">
                <span className={`px-2 py-1 text-xs border ${user.status === 'active' ? 'border-ochre text-ochre' : 'border-ink-light text-ink-light'}`}>
                  {getUserStatusLabel(language, user.status)}
                </span>
              </td>
              <td className="py-4 text-right space-x-4">
                {user.role !== 'admin' && (
                  <button onClick={() => handleStatus(user.id, user.status)} disabled={busyId === user.id} className="text-ochre opacity-80 hover:opacity-100 tracking-wider disabled:opacity-50 disabled:cursor-not-allowed">
                    {user.status === 'active' ? t('users.disable') : t('users.activate')}
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
