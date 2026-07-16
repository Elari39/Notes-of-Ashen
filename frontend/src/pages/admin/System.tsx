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
import {
  BACKUP_PASSPHRASE_MAX_LENGTH,
  BACKUP_PASSPHRASE_MIN_LENGTH,
  canStartBackupExport,
  canStartBackupRestore,
  countBackupPassphraseCharacters,
  hasValidBackupCredentials,
  isBackupPassphraseValid,
} from './backupPolicy';

const AdminSystem: React.FC = () => {
  const language = usePreferenceStore((state) => state.language);
  const logout = useAuthStore((state) => state.logout);
  const confirm = useConfirm();
  const t = (key: Parameters<typeof translate>[1]) => translate(language, key);
  const [health, setHealth] = useState<SystemHealth | null>(null);
  const [healthError, setHealthError] = useState('');
  const [backupError, setBackupError] = useState('');
  const [loading, setLoading] = useState(true);
  const [currentPassword, setCurrentPassword] = useState('');
  const [passphrase, setPassphrase] = useState('');
  const [confirmation, setConfirmation] = useState('');
  const [backupFile, setBackupFile] = useState<File | null>(null);
  const [busy, setBusy] = useState<'export' | 'restore' | null>(null);
  const busyRef = useRef<'export' | 'restore' | null>(null);

  const loadHealth = useCallback(async (refresh = false) => {
    setLoading(true);
    try {
      const response = await getSystemHealth(refresh);
      setHealth(response.data);
      setHealthError('');
    } catch (err) {
      setHealthError(getErrorMessage(err, translate(language, 'system.healthError')));
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

  const credentials = { currentPassword, passphrase };
  const passphraseLength = countBackupPassphraseCharacters(passphrase);
  const passphraseValid = isBackupPassphraseValid(passphrase);
  const validCredentials = hasValidBackupCredentials(credentials);
  const isBusy = busy !== null;
  const exportAvailable = canStartBackupExport({ credentials, busy: isBusy });
  const restoreAvailable = canStartBackupRestore({
    credentials,
    file: backupFile,
    confirmation,
    busy: isBusy,
  });

  const beginBackupOperation = (operation: 'export' | 'restore'): boolean => {
    if (busyRef.current !== null) return false;
    busyRef.current = operation;
    setBusy(operation);
    return true;
  };

  const finishBackupOperation = (operation: 'export' | 'restore') => {
    if (busyRef.current !== operation) return;
    busyRef.current = null;
    setBusy(null);
  };

  const handleExport = async () => {
    if (!exportAvailable || !beginBackupOperation('export')) return;
    setBackupError('');
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
      setBackupError(getErrorMessage(err, t('backup.exportError')));
    } finally {
      finishBackupOperation('export');
    }
  };

  const handleRestore = async () => {
    if (busyRef.current !== null) return;
    if (!backupFile) {
      setBackupError(t('backup.file'));
      return;
    }
    if (confirmation !== 'REPLACE') {
      setBackupError(t('backup.confirmationError'));
      return;
    }
    if (!validCredentials) return;
    const accepted = await confirm({
      title: t('backup.confirm'),
      description: t('backup.warning'),
      tone: 'danger',
    });
    if (!accepted) return;
    if (!beginBackupOperation('restore')) return;
    setBackupError('');
    try {
      await restoreBackup(backupFile, currentPassword, passphrase, confirmation);
      finishBackupOperation('restore');
      logout();
      window.location.assign('/login');
    } catch (err) {
      setBackupError(getErrorMessage(err, t('backup.restoreError')));
      finishBackupOperation('restore');
    }
  };

  const handleBackupFileChange = (event: React.ChangeEvent<HTMLInputElement>) => {
    const file = event.target.files?.[0] ?? null;
    if (!file) {
      setBackupFile(null);
      return;
    }
    if (!file.name.toLowerCase().endsWith('.noa-backup')) {
      event.target.value = '';
      setBackupFile(null);
      setBackupError(t('backup.fileInvalid'));
      return;
    }
    setBackupFile(file);
    setBackupError('');
  };

  const passphraseError = passphraseLength > 0 && !passphraseValid
    ? passphraseLength < BACKUP_PASSPHRASE_MIN_LENGTH
      ? formatText(t('backup.passphraseTooShort'), { min: BACKUP_PASSPHRASE_MIN_LENGTH })
      : formatText(t('backup.passphraseTooLong'), { max: BACKUP_PASSPHRASE_MAX_LENGTH })
    : '';

  return (
    <div>
      <header className="mb-8 border-b border-hairline pb-5">
        <p className="editorial-kicker">{t('admin.system')}</p>
        <h3 className="mt-3 text-4xl text-ink">{t('system.title')}</h3>
        <p className="mt-2 text-sm text-muted">{t('system.subtitle')}</p>
      </header>

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
        <InlineNotice message={healthError} className="mb-4" />

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
        <InlineNotice message={backupError} className="mt-4" />

        <div className="mt-5 grid gap-4 md:grid-cols-2">
          <div>
            <label htmlFor="backup-current-password" className="mb-2 block text-sm font-medium text-ink">
              {t('backup.currentPassword')}
            </label>
            <TextField
              id="backup-current-password"
              type="password"
              autoComplete="current-password"
              value={currentPassword}
              disabled={isBusy}
              aria-describedby="backup-current-password-hint"
              onChange={(event) => {
                setCurrentPassword(event.target.value);
                setBackupError('');
              }}
            />
            <p id="backup-current-password-hint" className="mt-2 text-xs text-muted">
              {t('backup.currentPasswordHint')}
            </p>
          </div>
          <div>
            <label htmlFor="backup-passphrase" className="mb-2 block text-sm font-medium text-ink">
              {t('backup.passphrase')}
            </label>
            <TextField
              id="backup-passphrase"
              type="password"
              autoComplete="new-password"
              value={passphrase}
              disabled={isBusy}
              invalid={passphraseLength > 0 && !passphraseValid}
              aria-describedby="backup-passphrase-hint"
              onChange={(event) => {
                setPassphrase(event.target.value);
                setBackupError('');
              }}
            />
            <p
              id="backup-passphrase-hint"
              aria-live="polite"
              className={`mt-2 text-xs ${passphraseError ? 'text-ember' : 'text-muted'}`}
            >
              {formatText(t('backup.passphraseLength'), {
                count: passphraseLength,
                min: BACKUP_PASSPHRASE_MIN_LENGTH,
                max: BACKUP_PASSPHRASE_MAX_LENGTH,
              })}
              {passphraseError ? ` ${passphraseError}` : ''}
            </p>
          </div>
        </div>
        <div className="mt-4 max-w-md">
          <label htmlFor="backup-confirmation" className="mb-2 block text-sm font-medium text-ink">
            {t('backup.confirmation')}
          </label>
          <TextField
            id="backup-confirmation"
            value={confirmation}
            disabled={isBusy}
            onChange={(event) => {
              setConfirmation(event.target.value);
              setBackupError('');
            }}
          />
        </div>

        <div className="mt-5 flex flex-wrap items-center gap-3">
          <Button loading={busy === 'export'} disabled={!exportAvailable} onClick={() => void handleExport()}>
            {busy === 'export' ? t('backup.exporting') : t('backup.export')}
          </Button>
          <div>
            <label htmlFor="backup-file" className="mb-1 block text-sm text-muted">{t('backup.file')}</label>
            <input
              id="backup-file"
              type="file"
              accept=".noa-backup"
              disabled={isBusy}
              className="max-w-full text-sm text-muted disabled:cursor-not-allowed disabled:opacity-50"
              onChange={handleBackupFileChange}
            />
            {backupFile && (
              <p className="mt-1 max-w-sm truncate text-xs text-muted" title={backupFile.name}>
                {formatText(t('backup.fileSelected'), { name: backupFile.name })}
              </p>
            )}
          </div>
          <Button
            variant="danger"
            loading={busy === 'restore'}
            disabled={!restoreAvailable}
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
