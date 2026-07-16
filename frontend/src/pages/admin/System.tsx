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
  getBackupExportBlockerReason,
  getBackupRestoreBlockerReason,
  isBackupPassphraseValid,
  type BackupBlockerReason,
} from './backupPolicy';

const backupBlockerMessageKey: Record<BackupBlockerReason, Parameters<typeof translate>[1]> = {
  busy: 'backup.blocker.busy',
  currentPasswordMissing: 'backup.blocker.currentPasswordMissing',
  passphraseMissing: 'backup.blocker.passphraseMissing',
  passphraseTooShort: 'backup.blocker.passphraseTooShort',
  passphraseTooLong: 'backup.blocker.passphraseTooLong',
  fileMissing: 'backup.blocker.fileMissing',
  confirmationMissing: 'backup.blocker.confirmationMissing',
  confirmationMismatch: 'backup.blocker.confirmationMismatch',
};

const DownloadIcon: React.FC = () => (
  <svg width="16" height="16" viewBox="0 0 16 16" fill="none" aria-hidden="true">
    <path d="M8 2.5V10M8 10L5 7M8 10L11 7M3 12.5H13" stroke="currentColor" strokeWidth="1.5" strokeLinecap="round" strokeLinejoin="round" />
  </svg>
);

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
  const [backupFileError, setBackupFileError] = useState('');
  const [busy, setBusy] = useState<'export' | 'restore' | null>(null);
  const busyRef = useRef<'export' | 'restore' | null>(null);
  const backupFileInputRef = useRef<HTMLInputElement>(null);

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
  const isBusy = busy !== null;
  const exportState = { credentials, busy: isBusy };
  const restoreState = {
    credentials,
    file: backupFile,
    confirmation,
    busy: isBusy,
  };
  const exportBlockerReason = getBackupExportBlockerReason(exportState);
  const restoreBlockerReason = getBackupRestoreBlockerReason(restoreState);
  const exportAvailable = canStartBackupExport(exportState);
  const restoreAvailable = canStartBackupRestore(restoreState);

  const getBlockerMessage = (reason: BackupBlockerReason | null): string => {
    if (!reason) return '';
    const message = t(backupBlockerMessageKey[reason]);
    if (reason === 'passphraseTooShort') {
      return formatText(message, { min: BACKUP_PASSPHRASE_MIN_LENGTH });
    }
    if (reason === 'passphraseTooLong') {
      return formatText(message, { max: BACKUP_PASSPHRASE_MAX_LENGTH });
    }
    return message;
  };

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
    if (getBackupExportBlockerReason({ credentials, busy: busyRef.current !== null }) || !beginBackupOperation('export')) return;
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
    if (getBackupRestoreBlockerReason({
      credentials,
      file: backupFile,
      confirmation,
      busy: busyRef.current !== null,
    })) return;
    if (!backupFile) return;
    const accepted = await confirm({
      title: t('backup.confirm'),
      description: t('backup.warning'),
      tone: 'danger',
    });
    if (!accepted) return;
    if (getBackupRestoreBlockerReason({
      credentials,
      file: backupFile,
      confirmation,
      busy: busyRef.current !== null,
    })) return;
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
    if (!file) return;
    if (!file.name.toLowerCase().endsWith('.noa-backup')) {
      event.target.value = '';
      setBackupFile(null);
      setBackupFileError(t('backup.fileInvalid'));
      return;
    }
    setBackupFile(file);
    setBackupFileError('');
  };

  const openBackupFilePicker = () => {
    if (isBusy) return;
    if (backupFileInputRef.current) {
      backupFileInputRef.current.value = '';
      backupFileInputRef.current.click();
    }
  };

  const removeBackupFile = () => {
    if (isBusy) return;
    setBackupFile(null);
    setBackupFileError('');
    if (backupFileInputRef.current) backupFileInputRef.current.value = '';
  };

  const passphraseError = passphraseLength > 0 && !passphraseValid
    ? passphraseLength < BACKUP_PASSPHRASE_MIN_LENGTH
      ? formatText(t('backup.passphraseTooShort'), { min: BACKUP_PASSPHRASE_MIN_LENGTH })
      : formatText(t('backup.passphraseTooLong'), { max: BACKUP_PASSPHRASE_MAX_LENGTH })
    : '';
  const confirmationInvalid = confirmation.length > 0 && confirmation !== 'REPLACE';
  const exportBlockerMessage = getBlockerMessage(exportBlockerReason);
  const restoreBlockerMessage = getBlockerMessage(restoreBlockerReason);

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
        <InlineNotice
          message={backupError}
          icon
          onDismiss={() => setBackupError('')}
          className="mt-4"
        />

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
        <div className="mt-6 grid gap-4 lg:grid-cols-2">
          <article className="flex flex-col rounded-lg border border-hairline bg-surface-card p-5">
            <div>
              <h5 className="text-base font-semibold text-ink">{t('backup.exportCardTitle')}</h5>
              <p className="mt-1 text-sm leading-6 text-muted">{t('backup.exportCardDescription')}</p>
            </div>
            <div className="mt-5 flex flex-1 flex-col justify-end gap-3">
              <Button
                variant="primary"
                size="lg"
                fullWidth
                iconBefore={<DownloadIcon />}
                loading={busy === 'export'}
                disabled={!exportAvailable}
                aria-describedby={exportBlockerReason ? 'backup-export-blocker' : undefined}
                onClick={() => void handleExport()}
              >
                {busy === 'export' ? t('backup.exporting') : t('backup.export')}
              </Button>
              {exportBlockerReason && (
                <p id="backup-export-blocker" role="status" aria-live="polite" className="text-xs leading-5 text-muted">
                  {exportBlockerMessage}
                </p>
              )}
            </div>
          </article>

          <article className="rounded-lg border border-ember/35 bg-[var(--ember-soft)] p-5">
            <div>
              <h5 className="text-base font-semibold text-ember">{t('backup.restoreCardTitle')}</h5>
              <p className="mt-1 text-sm leading-6 text-muted">{t('backup.restoreCardDescription')}</p>
            </div>

            <div className="mt-5">
              <input
                ref={backupFileInputRef}
                id="backup-file"
                type="file"
                accept=".noa-backup"
                disabled={isBusy}
                className="sr-only"
                aria-label={t('backup.file')}
                aria-describedby="backup-file-status"
                onChange={handleBackupFileChange}
              />
              <div className="flex flex-wrap items-center gap-2">
                <Button variant="ghost" size="sm" disabled={isBusy} aria-controls="backup-file" onClick={openBackupFilePicker}>
                  {backupFile ? t('backup.replaceFile') : t('backup.chooseFile')}
                </Button>
                {backupFile && (
                  <Button variant="subtle" size="sm" disabled={isBusy} onClick={removeBackupFile}>
                    {t('backup.removeFile')}
                  </Button>
                )}
              </div>
              <p
                id="backup-file-status"
                aria-live="polite"
                className={`mt-2 max-w-full truncate text-xs ${backupFileError ? 'text-ember' : 'text-muted'}`}
                title={backupFile?.name}
              >
                {backupFileError || (backupFile
                  ? formatText(t('backup.fileSelected'), { name: backupFile.name })
                  : t('backup.noFileSelected'))}
              </p>
            </div>

            <div className="mt-4">
              <label htmlFor="backup-confirmation" className="mb-2 block text-sm font-medium text-ink">
                {t('backup.confirmation')}
              </label>
              <TextField
                id="backup-confirmation"
                value={confirmation}
                disabled={isBusy}
                invalid={confirmationInvalid}
                aria-describedby="backup-confirmation-hint"
                onChange={(event) => {
                  setConfirmation(event.target.value);
                }}
              />
              <p
                id="backup-confirmation-hint"
                aria-live="polite"
                className={`mt-2 text-xs ${confirmationInvalid ? 'text-ember' : 'text-muted'}`}
              >
                {confirmationInvalid ? t('backup.confirmationError') : t('backup.confirmationHint')}
              </p>
            </div>

            <div className="mt-5 space-y-3">
              <Button
                variant="danger"
                size="lg"
                fullWidth
                loading={busy === 'restore'}
                disabled={!restoreAvailable}
                aria-describedby={restoreBlockerReason ? 'backup-restore-blocker' : undefined}
                onClick={() => void handleRestore()}
              >
                {busy === 'restore' ? t('backup.restoring') : t('backup.restore')}
              </Button>
              {restoreBlockerReason && (
                <p id="backup-restore-blocker" role="status" aria-live="polite" className="text-xs leading-5 text-ember">
                  {restoreBlockerMessage}
                </p>
              )}
            </div>
          </article>
        </div>
      </section>
    </div>
  );
};

export default AdminSystem;
