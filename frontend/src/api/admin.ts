import http from '../utils/http';
import type { AdminStats, BaseResp } from '../types';

export const getAdminStats = () =>
  http.get<unknown, BaseResp<AdminStats>>('/admin/stats');
