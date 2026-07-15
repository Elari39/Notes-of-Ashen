import http from '../utils/http';
import type { BackupRestoreResult, BaseResp, SystemHealth } from '../types';
import type { BackupExportReq } from '../types/api';

export const getSystemHealth = (refresh = false, signal?: AbortSignal) =>
  http.get<unknown, BaseResp<SystemHealth>>('/admin/system/health', { params: refresh ? { refresh: true } : undefined, signal });

export const exportBackup = (data: BackupExportReq) =>
  http.post('/admin/backups/export', data, { responseType: 'blob', timeout: 0 });

export const restoreBackup = (file: File, currentPassword: string, passphrase: string, confirmation: string) => {
  const data = new FormData(); data.append('file', file); data.append('currentPassword', currentPassword);
  data.append('passphrase', passphrase); data.append('confirmation', confirmation);
  return http.post<unknown, BaseResp<BackupRestoreResult>>('/admin/backups/restore', data, { timeout: 0 });
};
