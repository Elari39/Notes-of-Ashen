import assert from 'node:assert/strict';
import test from 'node:test';

import {
  canStartBackupRestore,
  canStartBackupExport,
  countBackupPassphraseCharacters,
  hasValidBackupCredentials,
  isBackupPassphraseValid,
} from '../src/pages/admin/backupPolicy.ts';

test('归档密码按 Unicode 码点而非 UTF-16 长度校验', () => {
  assert.equal(countBackupPassphraseCharacters('😀'.repeat(6)), 6);
  assert.equal(isBackupPassphraseValid('12345678901'), false);
  assert.equal(isBackupPassphraseValid('123456789012'), true);
  assert.equal(isBackupPassphraseValid('x'.repeat(129)), false);
  assert.equal(isBackupPassphraseValid('😀'.repeat(6)), false);
  assert.equal(isBackupPassphraseValid('😀'.repeat(12)), true);
  assert.equal(isBackupPassphraseValid('😀'.repeat(65)), true);
  assert.equal(isBackupPassphraseValid('😀'.repeat(129)), false);
});

test('恢复只在凭据、文件、确认词和空闲状态均满足时可开始', () => {
  const credentials = { currentPassword: 'current-password', passphrase: 'archive-pass1' };
  const file = {} as File;

  assert.equal(hasValidBackupCredentials(credentials), true);
  assert.equal(canStartBackupExport({ credentials, busy: true }), false);
  assert.equal(canStartBackupExport({ credentials, busy: false }), true);
  assert.equal(canStartBackupRestore({ credentials, file: null, confirmation: 'REPLACE', busy: false }), false);
  assert.equal(canStartBackupRestore({ credentials, file, confirmation: '', busy: false }), false);
  assert.equal(canStartBackupRestore({ credentials, file, confirmation: 'REPLACE', busy: true }), false);
  assert.equal(canStartBackupRestore({ credentials, file, confirmation: 'REPLACE', busy: false }), true);
});
