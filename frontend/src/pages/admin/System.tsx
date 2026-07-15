import React, { useCallback, useEffect, useRef, useState } from 'react';
import { exportBackup, getSystemHealth, restoreBackup } from '../../api/system';
import type { SystemHealth } from '../../types';
import { usePreferenceStore } from '../../store/preferences';
import { useAuthStore } from '../../store/auth';
import { formatText, translate } from '../../i18n';
import { getErrorMessage } from '../../utils/error';
import { useConfirm } from '../../hooks/useConfirm';
import InlineNotice from '../../components/InlineNotice';
import Button from '../../components/ui/Button';
import TextField from '../../components/ui/TextField';
import PagePendingState from '../../components/RoutePending';

const AdminSystem: React.FC = () => {
  const language = usePreferenceStore((state) => state.language);
  const logout = useAuthStore((state) => state.logout);
  const confirm = useConfirm();
  const t = (key: Parameters<typeof translate>[1]) => translate(language, key);
  const [health, setHealth] = useState<SystemHealth | null>(null);
  const [error, setError] = useState('');
  const [loading, setLoading] = useState(true);
  const [currentPassword, setCurrentPassword] = useState('');
  const [passphrase, setPassphrase] = useState('');
  const [confirmation, setConfirmation] = useState('');
  const [busy, setBusy] = useState<'export' | 'restore' | null>(null);
  const fileRef = useRef<HTMLInputElement>(null);

  const loadHealth = useCallback(async (refresh = false) => {
    setLoading(true);
    try {
      const response = await getSystemHealth(refresh);
      setHealth(response.data);
      setError('');
    } catch (err) {
      setError(getErrorMessage(err, translate(language, 'system.healthError')));
    } finally {
      setLoading(false);
    }
  }, [language]);

  useEffect(() => {
    void loadHealth();
    const timer = window.setInterval(() => {
      if (document.visibilityState === 'visible') {
        void loadHealth();
      }
    }, 30_000);
    return () => window.clearInterval(timer);
  }, [loadHealth]);

  const handleExport = async () => {
    setBusy('export');
    setError('');
    try {
      const response = await exportBackup({ currentPassword, passphrase });
      const url = URL.createObjectURL(response.data as Blob);
      const anchor = document.createElement('a');
      anchor.href = url;
      anchor.download = `notes-of-ashen-${new Date().toISOString().replace(/[:.]/g, '-')}.noa-backup`;
      document.body.appendChild(anchor);
      anchor.click();
      anchor.remove();
      window.setTimeout(() => URL.revokeObjectURL(url), 0);
      setCurrentPassword('');
      setPassphrase('');
    } catch (err) {
      setError(getErrorMessage(err, t('backup.exportError')));
    } finally {
      setBusy(null);
    }
  };

  const handleRestore = async () => {
    const file = fileRef.current?.files?.[0];
    if (!file) {
      setError(t('backup.file'));
      return;
    }
    if (confirmation !== 'REPLACE') {
      setError(t('backup.confirmationError'));
      return;
    }
    const accepted = await confirm({
      title: t('backup.confirm'),
      description: t('backup.warning'),
      tone: 'danger',
    });
    if (!accepted) return;
    setBusy('restore');
    setError('');
    try {
      await restoreBackup(file, currentPassword, passphrase, confirmation);
      logout();
      window.location.assign('/login');
    } catch (err) {
      setError(getErrorMessage(err, t('backup.restoreError')));
      setBusy(null);
    }
  };

  const validCredentials = currentPassword.length > 0 && passphrase.length >= 12 && passphrase.length <= 128;

  return (
    <div>
      <header className="mb-8 border-b border-hairline pb-5">
        <p className="editorial-kicker">{t('admin.system')}</p>
        <h3 className="mt-3 text-4xl text-ink">{t('system.title')}</h3>
        <p className="mt-2 text-sm text-muted">{t('system.subtitle')}</p>
      </header>
      <InlineNotice message={error} className="mb-5" />

      <section className="mb-8 rounded-lg bg-paper p-5 shadow-xs">
        <div className="mb-4 flex items-center justify-between gap-4">
          <div>
            <h4 className="font-display text-2xl text-ink">{t('system.health')}</h4>
            {health && (
              <p className="mt-1 text-xs text-muted">
                {health.status === 'healthy' ? t('system.overallHealthy') : t('system.overallDegraded')}
                {' · '}
                {formatText(t('system.checkedAt'), { time: new Date(health.checkedAt).toLocaleString() })}
              </p>
            )}
          </div>
          <Button size="sm" loading={loading} onClick={() => void loadHealth(true)}>{t('system.refresh')}</Button>
        </div>

        {loading && !health ? (
          <PagePendingState variant="inline" label={t('common.loading')} />
        ) : health ? (
          <div className="grid gap-3 sm:grid-cols-2 lg:grid-cols-3">
            {health.checks.map((check) => (
              <div key={check.name} className="rounded-md border border-hairline p-4">
                <div className="flex items-center justify-between">
                  <span className="font-medium text-ink">{check.name}</span>
                  <span className={check.status === 'up' ? 'text-moss' : check.status === 'down' ? 'text-ember' : 'text-muted'}>
                    {t(`system.status.${check.status}`)}
                  </span>
                </div>
                <p className="mt-2 text-xs text-muted">
                  {check.status === 'disabled' ? t('system.disabledDetail') : `${check.latencyMs} ms`}
                </p>
              </div>
            ))}
          </div>
        ) : null}
      </section>

      <section className="rounded-lg bg-paper p-5 shadow-xs">
        <h4 className="font-display text-2xl text-ink">{t('backup.title')}</h4>
        <p className="mt-2 text-sm leading-6 text-ember">{t('backup.warning')}</p>
        <ul className="mt-3 list-disc space-y-1 pl-5 text-sm text-muted">
          <li>{t('backup.warningReplace')}</li>
          <li>{t('backup.warningStats')}</li>
          <li>{t('backup.warningAI')}</li>
          <li>{t('backup.warningLogout')}</li>
        </ul>

        <div className="mt-5 grid gap-4 md:grid-cols-2">
          <TextField type="password" value={currentPassword} onChange={(event) => setCurrentPassword(event.target.value)} placeholder={t('backup.currentPassword')} />
          <TextField type="password" value={passphrase} onChange={(event) => setPassphrase(event.target.value)} placeholder={t('backup.passphrase')} />
        </div>
        <div className="mt-4 max-w-md">
          <TextField value={confirmation} onChange={(event) => setConfirmation(event.target.value)} placeholder={t('backup.confirmation')} />
        </div>

        <div className="mt-5 flex flex-wrap items-center gap-3">
          <Button loading={busy === 'export'} disabled={!validCredentials} onClick={() => void handleExport()}>
            {busy === 'export' ? t('backup.exporting') : t('backup.export')}
          </Button>
          <label className="text-sm text-muted">
            <span className="sr-only">{t('backup.file')}</span>
            <input ref={fileRef} type="file" accept=".noa-backup" className="max-w-full text-sm text-muted" />
          </label>
          <Button
            variant="danger"
            loading={busy === 'restore'}
            disabled={!validCredentials || confirmation !== 'REPLACE'}
            onClick={() => void handleRestore()}
          >
            {busy === 'restore' ? t('backup.restoring') : t('backup.restore')}
          </Button>
        </div>
      </section>
    </div>
  );
};

export default AdminSystem;
