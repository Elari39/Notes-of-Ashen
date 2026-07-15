import type {
  AIAPIFormat,
  AIConnectionReq,
  AIModelTestReq,
  AISettingsResp,
  UpdateAISettingsReq,
} from '../../types/api';

export type AISettingsOperation = 'save' | 'models' | 'test';

export interface AISettingsDraft {
  enabled: boolean;
  apiFormat: AIAPIFormat;
  baseUrl: string;
  model: string;
  apiKey: string;
  clearApiKey: boolean;
  firstByteTimeoutSeconds: number;
  nonStreamTimeoutSeconds: number;
}

export const toAISettingsDraft = (settings: AISettingsResp): AISettingsDraft => ({
  enabled: Boolean(settings.enabled),
  apiFormat: settings.apiFormat === 'anthropic' ? 'anthropic' : 'openai',
  baseUrl: settings.baseUrl || '',
  model: settings.model || '',
  apiKey: '',
  clearApiKey: false,
  firstByteTimeoutSeconds: settings.firstByteTimeoutSeconds ?? 60,
  nonStreamTimeoutSeconds: settings.nonStreamTimeoutSeconds ?? 600,
});

export const isAISettingsTimeoutInvalid = (draft: AISettingsDraft): boolean => (
  draft.firstByteTimeoutSeconds < 1 ||
  draft.firstByteTimeoutSeconds > 1800 ||
  draft.nonStreamTimeoutSeconds < draft.firstByteTimeoutSeconds ||
  draft.nonStreamTimeoutSeconds > 1800
);

export const isAISettingsDirty = (
  draft: AISettingsDraft,
  baseline: AISettingsDraft | null,
): boolean => {
  if (!baseline) {
    return false;
  }

  return (
    draft.enabled !== baseline.enabled ||
    draft.apiFormat !== baseline.apiFormat ||
    draft.baseUrl.trim() !== baseline.baseUrl.trim() ||
    draft.model.trim() !== baseline.model.trim() ||
    draft.apiKey.trim() !== '' ||
    draft.clearApiKey !== baseline.clearApiKey ||
    draft.firstByteTimeoutSeconds !== baseline.firstByteTimeoutSeconds ||
    draft.nonStreamTimeoutSeconds !== baseline.nonStreamTimeoutSeconds
  );
};

export const hasAIConnectionIdentityChanged = (
  previous: AISettingsDraft,
  next: AISettingsDraft,
): boolean => (
  previous.apiFormat !== next.apiFormat ||
  previous.baseUrl.trim() !== next.baseUrl.trim() ||
  previous.apiKey !== next.apiKey ||
  previous.clearApiKey !== next.clearApiKey
);

const optionalAPIKey = (apiKey: string): Pick<AIConnectionReq, 'apiKey'> => {
  const normalized = apiKey.trim();
  return normalized ? { apiKey: normalized } : {};
};

export const buildAIConnectionReq = (draft: AISettingsDraft): AIConnectionReq => ({
  apiFormat: draft.apiFormat,
  baseUrl: draft.baseUrl.trim(),
  ...optionalAPIKey(draft.apiKey),
  firstByteTimeoutSeconds: draft.firstByteTimeoutSeconds,
  nonStreamTimeoutSeconds: draft.nonStreamTimeoutSeconds,
});

export const buildAIModelTestReq = (draft: AISettingsDraft): AIModelTestReq => ({
  ...buildAIConnectionReq(draft),
  model: draft.model.trim(),
});

export const buildUpdateAISettingsReq = (draft: AISettingsDraft): UpdateAISettingsReq => ({
  enabled: draft.enabled,
  apiFormat: draft.apiFormat,
  baseUrl: draft.baseUrl.trim(),
  model: draft.model.trim(),
  ...optionalAPIKey(draft.apiKey),
  clearApiKey: draft.clearApiKey,
  firstByteTimeoutSeconds: draft.firstByteTimeoutSeconds,
  nonStreamTimeoutSeconds: draft.nonStreamTimeoutSeconds,
});

export const normalizeAIModels = (models: string[]): string[] => (
  [...new Set(models.map((model) => model.trim()).filter(Boolean))]
);

export const canStartAISettingsOperation = (
  activeOperation: AISettingsOperation | null,
): boolean => activeOperation === null;
