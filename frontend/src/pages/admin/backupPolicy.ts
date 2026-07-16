export const BACKUP_PASSPHRASE_MIN_LENGTH = 12;
export const BACKUP_PASSPHRASE_MAX_LENGTH = 128;

export type BackupCredentialsState = {
  currentPassword: string;
  passphrase: string;
};

export const countBackupPassphraseCharacters = (value: string): number => Array.from(value).length;

export const isBackupPassphraseValid = (value: string): boolean => {
  const length = countBackupPassphraseCharacters(value);
  return length >= BACKUP_PASSPHRASE_MIN_LENGTH && length <= BACKUP_PASSPHRASE_MAX_LENGTH;
};

export const hasValidBackupCredentials = ({ currentPassword, passphrase }: BackupCredentialsState): boolean => (
  currentPassword.length > 0 && isBackupPassphraseValid(passphrase)
);

export const canStartBackupExport = ({
  credentials,
  busy,
}: {
  credentials: BackupCredentialsState;
  busy: boolean;
}): boolean => !busy && hasValidBackupCredentials(credentials);

export const canStartBackupRestore = ({
  credentials,
  file,
  confirmation,
  busy,
}: {
  credentials: BackupCredentialsState;
  file: File | null;
  confirmation: string;
  busy: boolean;
}): boolean => (
  !busy && hasValidBackupCredentials(credentials) && file !== null && confirmation === 'REPLACE'
);
