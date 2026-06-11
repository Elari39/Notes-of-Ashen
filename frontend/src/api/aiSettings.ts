import http from '../utils/http';
import type { BaseResp } from '../types';
import type { AISettingsResp, UpdateAISettingsReq } from '../types/api';

export const getAISettings = () =>
  http.get<unknown, BaseResp<AISettingsResp>>('/admin/ai/settings');

export const updateAISettings = (data: UpdateAISettingsReq) =>
  http.put<unknown, BaseResp<AISettingsResp>>('/admin/ai/settings', data);
