export const BACKUP_PASSPHRASE_MIN_LENGTH = 12;
export const BACKUP_PASSPHRASE_MAX_LENGTH = 128;

export type BackupCredentialsState = {
  currentPassword: string;
  passphrase: string;
};

export type BackupBlockerReason =
  | 'busy'
  | 'currentPasswordMissing'
  | 'passphraseMissing'
  | 'passphraseTooShort'
  | 'passphraseTooLong'
  | 'fileMissing'
  | 'confirmationMissing'
  | 'confirmationMismatch';

type BackupOperationState = {
  credentials: BackupCredentialsState;
  busy: boolean;
};

type BackupRestoreOperationState = BackupOperationState & {
  file: File | null;
  confirmation: string;
};

export const countBackupPassphraseCharacters = (value: string): number => Array.from(value).length;

export const isBackupPassphraseValid = (value: string): boolean => {
  const length = countBackupPassphraseCharacters(value);
  return length >= BACKUP_PASSPHRASE_MIN_LENGTH && length <= BACKUP_PASSPHRASE_MAX_LENGTH;
};

export const hasValidBackupCredentials = ({ currentPassword, passphrase }: BackupCredentialsState): boolean => (
  currentPassword.length > 0 && isBackupPassphraseValid(passphrase)
);

const getCredentialsBlockerReason = ({ currentPassword, passphrase }: BackupCredentialsState): BackupBlockerReason | null => {
  if (currentPassword.length === 0) return 'currentPasswordMissing';
  if (passphrase.length === 0) return 'passphraseMissing';

  const passphraseLength = countBackupPassphraseCharacters(passphrase);
  if (passphraseLength < BACKUP_PASSPHRASE_MIN_LENGTH) return 'passphraseTooShort';
  if (passphraseLength > BACKUP_PASSPHRASE_MAX_LENGTH) return 'passphraseTooLong';

  return null;
};

/**
 * 返回阻止备份操作的首个原因。优先级与界面提示保持一致，避免按钮禁用却没有解释。
 */
export const getBackupExportBlockerReason = ({ credentials, busy }: BackupOperationState): BackupBlockerReason | null => {
  if (busy) return 'busy';
  return getCredentialsBlockerReason(credentials);
};

export const getBackupRestoreBlockerReason = ({
  credentials,
  file,
  confirmation,
  busy,
}: BackupRestoreOperationState): BackupBlockerReason | null => {
  const exportBlocker = getBackupExportBlockerReason({ credentials, busy });
  if (exportBlocker) return exportBlocker;
  if (file === null) return 'fileMissing';
  if (confirmation.length === 0) return 'confirmationMissing';
  if (confirmation !== 'REPLACE') return 'confirmationMismatch';

  return null;
};

export const canStartBackupExport = (state: BackupOperationState): boolean => (
  getBackupExportBlockerReason(state) === null
);

export const canStartBackupRestore = (state: BackupRestoreOperationState): boolean => (
  getBackupRestoreBlockerReason(state) === null
);
