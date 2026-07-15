import http from '../utils/http';
import type { BaseResp } from '../types';
import type {
  AIConnectionReq,
  AIModelTestReq,
  AIModelTestResp,
  AIModelsResp,
  AISettingsResp,
  UpdateAISettingsReq,
} from '../types/api';

export const getAISettings = () =>
  http.get<unknown, BaseResp<AISettingsResp>>('/admin/ai/settings');

export const updateAISettings = (data: UpdateAISettingsReq) =>
  http.put<unknown, BaseResp<AISettingsResp>>('/admin/ai/settings', data);

export const getAIModels = (data: AIConnectionReq) =>
  http.post<unknown, BaseResp<AIModelsResp>>('/admin/ai/models', data);

export const testAIModel = (data: AIModelTestReq) =>
  http.post<unknown, BaseResp<AIModelTestResp>>('/admin/ai/test', data);
