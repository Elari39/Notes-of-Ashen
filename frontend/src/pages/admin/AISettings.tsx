import React, { useCallback, useEffect, useId, useMemo, useRef, useState } from 'react';
import InlineNotice from '../../components/InlineNotice';
import PagePendingState from '../../components/RoutePending';
import SettingsCard, { SettingsActions } from '../../components/admin/SettingsCard';
import Switch from '../../components/ui/Switch';
import Button from '../../components/ui/Button';
import { getAIModels, getAISettings, testAIModel, updateAISettings } from '../../api/aiSettings';
import { getErrorMessage } from '../../utils/error';
import { usePreferenceStore } from '../../store/preferences';
import { formatText, translate } from '../../i18n';
import type { AIAPIFormat, AISettingsResp } from '../../types/api';
import { canSaveAISettings, executeAISettingsUpdate } from './editorAccessPolicy';
import {
  buildAIConnectionReq,
  buildAIModelTestReq,
  buildUpdateAISettingsReq,
  canStartAISettingsOperation,
  isAISettingsDirty,
  isAISettingsTimeoutInvalid,
  normalizeAIModels,
  toAISettingsDraft,
  type AISettingsDraft,
  type AISettingsOperation,
} from './aiSettingsPolicy';

const AdminAISettings: React.FC = () => {
  const language = usePreferenceStore((state) => state.language);
  const t = (key: Parameters<typeof translate>[1]) => translate(language, key);
  const languageRef = useRef(language);
  languageRef.current = language;
  const modelInputId = useId();
  const apiKeyInputId = useId();

  const [draft, setDraft] = useState<AISettingsDraft | null>(null);
  const [baseline, setBaseline] = useState<AISettingsDraft | null>(null);
  const [apiKeyConfigured, setApiKeyConfigured] = useState(false);
  const [apiKeyNeedsUpdate, setApiKeyNeedsUpdate] = useState(false);
  const [modelOptions, setModelOptions] = useState<string[]>([]);
  const [loading, setLoading] = useState(true);
  const [hasLoaded, setHasLoaded] = useState(false);
  const [activeOperation, setActiveOperation] = useState<AISettingsOperation | null>(null);
  const [error, setError] = useState('');
  const [notice, setNotice] = useState('');
  const [modelsError, setModelsError] = useState('');
  const [modelsNotice, setModelsNotice] = useState('');
  const [testError, setTestError] = useState('');
  const [testNotice, setTestNotice] = useState('');
  const loadRequestIdRef = useRef(0);
  const activeOperationRef = useRef<AISettingsOperation | null>(null);

  const operationBusy = activeOperation !== null;
  const dirty = useMemo(() => (
    draft ? isAISettingsDirty(draft, baseline) : false
  ), [baseline, draft]);
  const timeoutInvalid = useMemo(() => (
    draft ? isAISettingsTimeoutInvalid(draft) : false
  ), [draft]);

  const resetConnectionFeedback = useCallback(() => {
    setModelOptions([]);
    setModelsError('');
    setModelsNotice('');
    setTestError('');
    setTestNotice('');
  }, []);

  const applyRemoteSettings = useCallback((settings: AISettingsResp) => {
    const nextDraft = toAISettingsDraft(settings);
    setDraft(nextDraft);
    setBaseline(nextDraft);
    setApiKeyConfigured(Boolean(settings.apiKeyConfigured));
    setApiKeyNeedsUpdate(Boolean(settings.apiKeyNeedsUpdate));
    resetConnectionFeedback();
  }, [resetConnectionFeedback]);

  const loadSettings = useCallback(async () => {
    const requestId = loadRequestIdRef.current + 1;
    loadRequestIdRef.current = requestId;
    setLoading(true);
    setHasLoaded(false);
    setError('');
    setNotice('');
    try {
      const res = await getAISettings();
      if (requestId !== loadRequestIdRef.current) {
        return;
      }
      applyRemoteSettings(res.data);
      setHasLoaded(true);
    } catch (requestError) {
      if (requestId !== loadRequestIdRef.current) {
        return;
      }
      setDraft(null);
      setBaseline(null);
      setError(getErrorMessage(requestError, translate(languageRef.current, 'aiSettings.loadError')));
    } finally {
      if (requestId === loadRequestIdRef.current) {
        setLoading(false);
      }
    }
  }, [applyRemoteSettings]);

  useEffect(() => {
    void loadSettings();
    return () => {
      loadRequestIdRef.current += 1;
    };
  }, [loadSettings]);

  const updateDraft = (patch: Partial<AISettingsDraft>) => {
    setDraft((current) => current ? { ...current, ...patch } : current);
    setError('');
    setNotice('');
  };

  const startOperation = (operation: AISettingsOperation): boolean => {
    if (!canStartAISettingsOperation(activeOperationRef.current)) {
      return false;
    }
    activeOperationRef.current = operation;
    setActiveOperation(operation);
    return true;
  };

  const finishOperation = () => {
    activeOperationRef.current = null;
    setActiveOperation(null);
  };

  const updateConnectionDraft = (patch: Partial<AISettingsDraft>) => {
    updateDraft(patch);
    resetConnectionFeedback();
  };

  const handleFormatChange = (apiFormat: AIAPIFormat) => {
    if (!draft || draft.apiFormat === apiFormat) {
      return;
    }
    updateConnectionDraft({ apiFormat });
  };

  const handleModelChange = (model: string) => {
    updateDraft({ model });
    setTestError('');
    setTestNotice('');
  };

  const handleAPIKeyChange = (apiKey: string) => {
    updateConnectionDraft({
      apiKey,
      clearApiKey: apiKey.trim() ? false : draft?.clearApiKey ?? false,
    });
  };

  const handleClearAPIKeyChange = (clearApiKey: boolean) => {
    updateConnectionDraft({ clearApiKey });
  };

  const handleCancel = () => {
    if (!baseline || operationBusy) {
      return;
    }
    setDraft({ ...baseline });
    setError('');
    setNotice('');
    resetConnectionFeedback();
  };

  const handleSubmit = async (event: React.FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    setError('');
    setNotice('');
    if (!draft || !canSaveAISettings(hasLoaded)) {
      setError(t('aiSettings.loadError'));
      return;
    }
    if (timeoutInvalid) {
      setError(t('aiSettings.timeoutError'));
      return;
    }
    if (!startOperation('save')) {
      return;
    }

    try {
      const res = await executeAISettingsUpdate(hasLoaded, () => (
        updateAISettings(buildUpdateAISettingsReq(draft))
      ));
      applyRemoteSettings(res.data);
      setNotice(t('aiSettings.saved'));
    } catch (requestError) {
      setError(getErrorMessage(requestError, t('aiSettings.saveError')));
    } finally {
      finishOperation();
    }
  };

  const handleGetModels = async () => {
    if (
      !draft ||
      draft.clearApiKey ||
      !draft.baseUrl.trim() ||
      timeoutInvalid
    ) {
      return;
    }
    if (!startOperation('models')) {
      return;
    }

    setModelsError('');
    setModelsNotice('');
    try {
      const res = await getAIModels(buildAIConnectionReq(draft));
      const models = normalizeAIModels(res.data.models || []);
      setModelOptions(models);
      setModelsNotice(models.length === 0
        ? t('aiSettings.modelsEmpty')
        : formatText(t('aiSettings.modelsLoaded'), { count: models.length }));
    } catch (requestError) {
      setModelsError(getErrorMessage(requestError, t('aiSettings.modelsError')));
    } finally {
      finishOperation();
    }
  };

  const handleTestModel = async () => {
    if (
      !draft ||
      draft.clearApiKey ||
      !draft.baseUrl.trim() ||
      !draft.model.trim() ||
      timeoutInvalid
    ) {
      return;
    }
    if (!startOperation('test')) {
      return;
    }

    setTestError('');
    setTestNotice('');
    try {
      const res = await testAIModel(buildAIModelTestReq(draft));
      setTestNotice(formatText(t('aiSettings.testSuccess'), {
        model: res.data.model,
        latency: res.data.latencyMs,
      }));
    } catch (requestError) {
      setTestError(getErrorMessage(requestError, t('aiSettings.testError')));
    } finally {
      finishOperation();
    }
  };

  return (
    <div>
      <div className="mb-8 border-b border-mountain-grey pb-4">
        <h3 className="text-2xl font-bold tracking-widest text-ink">{t('aiSettings.title')}</h3>
      </div>

      {hasLoaded && <InlineNotice message={error} className="mb-6" />}
      {hasLoaded && <InlineNotice message={notice} tone="success" className="mb-6" />}
      {loading && <PagePendingState variant="admin" label={t('aiSettings.loading')} />}
      {!loading && !hasLoaded && (
        <InlineNotice
          message={error || t('aiSettings.loadError')}
          className="mb-6"
          action={(
            <Button size="sm" onClick={() => void loadSettings()}>
              {t('common.retry')}
            </Button>
          )}
        />
      )}

      {hasLoaded && draft && (
        <form onSubmit={handleSubmit} className="space-y-8">
          <fieldset disabled={operationBusy} className="space-y-8 disabled:opacity-60">
            <SettingsCard
              title={t('aiSettings.enableTitle')}
              description={t('aiSettings.enableDesc')}
              action={(
                <div className="flex items-center gap-3">
                  <Switch
                    checked={draft.enabled}
                    onCheckedChange={(enabled) => updateDraft({ enabled })}
                    disabled={operationBusy}
                    label={t('aiSettings.enableTitle')}
                  />
                  <span className="text-xs tracking-widest text-ink-light">
                    {draft.enabled ? t('settings.enabled') : t('settings.disabled')}
                  </span>
                </div>
              )}
            />

            <SettingsCard
              title={t('aiSettings.providerTitle')}
              description={t('aiSettings.providerDesc')}
            >
              <div className="grid grid-cols-1 gap-5 md:grid-cols-2">
                <div className="md:col-span-2">
                  <span className="mb-2 block text-sm tracking-widest text-ink-light">{t('aiSettings.apiFormat')}</span>
                  <div className="grid grid-cols-1 border border-mountain-grey sm:grid-cols-2" role="radiogroup" aria-label={t('aiSettings.apiFormat')}>
                    <FormatOption
                      value="openai"
                      selected={draft.apiFormat === 'openai'}
                      disabled={operationBusy}
                      title={t('aiSettings.openaiFormat')}
                      description={t('aiSettings.openaiFormatDesc')}
                      onSelect={handleFormatChange}
                    />
                    <FormatOption
                      value="anthropic"
                      selected={draft.apiFormat === 'anthropic'}
                      disabled={operationBusy}
                      title={t('aiSettings.anthropicFormat')}
                      description={t('aiSettings.anthropicFormatDesc')}
                      onSelect={handleFormatChange}
                      className="border-t border-mountain-grey sm:border-l sm:border-t-0"
                    />
                  </div>
                </div>

                <label className="block text-sm text-ink-light">
                  <span className="mb-2 block tracking-widest">Base URL</span>
                  <input
                    value={draft.baseUrl}
                    onChange={(event) => updateConnectionDraft({ baseUrl: event.target.value })}
                    disabled={operationBusy}
                    placeholder={draft.apiFormat === 'anthropic' ? 'https://api.anthropic.com' : 'https://api.openai.com/v1'}
                    className="w-full border border-mountain-grey bg-transparent px-3 py-2 text-ink outline-none focus:border-ochre disabled:cursor-not-allowed disabled:opacity-50"
                  />
                </label>

                <div className="block text-sm text-ink-light">
                  <label htmlFor={modelInputId} className="mb-2 block tracking-widest">Model</label>
                  <div className="flex flex-col gap-2 sm:flex-row">
                    <input
                      id={modelInputId}
                      value={draft.model}
                      onChange={(event) => handleModelChange(event.target.value)}
                      disabled={operationBusy}
                      placeholder={t('aiSettings.modelPlaceholder')}
                      autoComplete="off"
                      className="min-w-0 flex-1 border border-mountain-grey bg-transparent px-3 py-2 text-ink outline-none focus:border-ochre disabled:cursor-not-allowed disabled:opacity-50"
                    />
                    {modelOptions.length > 0 && (
                      <select
                        value=""
                        onChange={(event) => handleModelChange(event.target.value)}
                        disabled={operationBusy}
                        aria-label={t('aiSettings.modelCandidatesLabel')}
                        className="w-full border border-mountain-grey bg-transparent px-3 py-2 text-ink outline-none focus:border-ochre disabled:cursor-not-allowed disabled:opacity-50 sm:w-48"
                      >
                        <option value="" disabled>
                          {formatText(t('aiSettings.modelCandidates'), { count: modelOptions.length })}
                        </option>
                        {modelOptions.map((option) => <option key={option} value={option}>{option}</option>)}
                      </select>
                    )}
                  </div>
                </div>

                <div className="block text-sm text-ink-light md:col-span-2">
                  <label htmlFor={apiKeyInputId} className="mb-2 block tracking-widest">
                    {t('aiSettings.apiKey')}
                  </label>
                  <input
                    id={apiKeyInputId}
                    value={draft.apiKey}
                    onChange={(event) => handleAPIKeyChange(event.target.value)}
                    disabled={operationBusy || draft.clearApiKey}
                    type="password"
                    autoComplete="new-password"
                    placeholder={apiKeyConfigured ? t('aiSettings.apiKeyConfigured') : t('aiSettings.apiKeyPlaceholder')}
                    className="w-full border border-mountain-grey bg-transparent px-3 py-2 text-ink outline-none focus:border-ochre disabled:cursor-not-allowed disabled:opacity-50"
                  />
                  <div className="mt-3 flex flex-wrap items-center gap-3 text-xs text-ink-light">
                    <span>{apiKeyConfigured ? t('aiSettings.configured') : t('aiSettings.notConfigured')}</span>
                    {apiKeyConfigured && (
                      <label className="inline-flex items-center gap-2">
                        <input
                          type="checkbox"
                          checked={draft.clearApiKey}
                          disabled={operationBusy || Boolean(draft.apiKey.trim())}
                          onChange={(event) => handleClearAPIKeyChange(event.target.checked)}
                          className="h-4 w-4 accent-ochre"
                        />
                        <span>{t('aiSettings.clearKey')}</span>
                      </label>
                    )}
                  </div>
                </div>
              </div>

              {apiKeyNeedsUpdate && (
                <InlineNotice message={t('aiSettings.keyNeedsUpdate')} tone="warning" icon className="mt-5" />
              )}

              <SettingsActions className="mt-5 border-t border-mountain-grey pt-5">
                <Button
                  type="button"
                  variant="ghost"
                  size="md"
                  disabled={operationBusy || draft.clearApiKey || timeoutInvalid || !draft.baseUrl.trim()}
                  loading={activeOperation === 'models'}
                  onClick={() => void handleGetModels()}
                >
                  {activeOperation === 'models' ? t('aiSettings.modelsLoading') : t('aiSettings.getModels')}
                </Button>
                <Button
                  type="button"
                  variant="ghost"
                  size="md"
                  disabled={operationBusy || draft.clearApiKey || timeoutInvalid || !draft.baseUrl.trim() || !draft.model.trim()}
                  loading={activeOperation === 'test'}
                  onClick={() => void handleTestModel()}
                >
                  {activeOperation === 'test' ? t('aiSettings.testing') : t('aiSettings.test')}
                </Button>
              </SettingsActions>
              <InlineNotice message={modelsError} className="mt-4" />
              <InlineNotice message={modelsNotice} tone={modelOptions.length > 0 ? 'success' : 'info'} icon className="mt-4" />
              <InlineNotice message={testError} className="mt-4" />
              <InlineNotice message={testNotice} tone="success" icon className="mt-4" />
            </SettingsCard>

            <SettingsCard
              title={t('aiSettings.timeoutTitle')}
              description={t('aiSettings.timeoutDesc')}
            >
              <div className="grid grid-cols-1 gap-5 md:grid-cols-2">
                <TimeoutInput
                  label={t('aiSettings.firstByte')}
                  value={draft.firstByteTimeoutSeconds}
                  disabled={operationBusy}
                  onChange={(firstByteTimeoutSeconds) => updateDraft({ firstByteTimeoutSeconds })}
                />
                <TimeoutInput
                  label={t('aiSettings.nonStream')}
                  value={draft.nonStreamTimeoutSeconds}
                  disabled={operationBusy}
                  onChange={(nonStreamTimeoutSeconds) => updateDraft({ nonStreamTimeoutSeconds })}
                />
              </div>
              {timeoutInvalid && (
                <p className="mt-3 text-sm text-ochre">{t('aiSettings.timeoutError')}</p>
              )}
            </SettingsCard>

            <SettingsActions>
              <Button
                type="submit"
                variant="primary"
                size="md"
                disabled={operationBusy || timeoutInvalid || !dirty}
                loading={activeOperation === 'save'}
              >
                {activeOperation === 'save' ? t('aiSettings.saving') : t('aiSettings.save')}
              </Button>
              <Button
                type="button"
                variant="ghost"
                size="md"
                disabled={operationBusy || !dirty}
                onClick={handleCancel}
              >
                {t('common.cancel')}
              </Button>
            </SettingsActions>
          </fieldset>
        </form>
      )}
    </div>
  );
};

type FormatOptionProps = {
  value: AIAPIFormat;
  selected: boolean;
  disabled: boolean;
  title: string;
  description: string;
  onSelect: (value: AIAPIFormat) => void;
  className?: string;
};

const FormatOption = ({
  value,
  selected,
  disabled,
  title,
  description,
  onSelect,
  className = '',
}: FormatOptionProps) => (
  <button
    type="button"
    role="radio"
    aria-checked={selected}
    disabled={disabled}
    onClick={() => onSelect(value)}
    className={`px-4 py-4 text-left transition-colors disabled:cursor-not-allowed disabled:opacity-50 ${
      selected ? 'bg-ink text-paper' : 'text-ink-light hover:text-ochre'
    } ${className}`.trim()}
  >
    <span className="block text-sm font-bold tracking-widest">{title}</span>
    <span className="mt-2 block text-xs leading-relaxed opacity-75">{description}</span>
  </button>
);

type TimeoutInputProps = {
  label: string;
  value: number;
  disabled: boolean;
  onChange: (value: number) => void;
};

const TimeoutInput = ({ label, value, disabled, onChange }: TimeoutInputProps) => (
  <label className="block text-sm text-ink-light">
    <span className="mb-2 block tracking-widest">{label}</span>
    <input
      type="number"
      min={1}
      max={1800}
      step={1}
      value={value}
      onChange={(event) => onChange(clampTimeout(event.target.value))}
      disabled={disabled}
      className="w-full border border-mountain-grey bg-transparent px-3 py-2 text-ink outline-none focus:border-ochre disabled:cursor-not-allowed disabled:opacity-50"
    />
  </label>
);

const clampTimeout = (value: string) => {
  const parsed = Number.parseInt(value, 10);
  if (!Number.isFinite(parsed)) {
    return 1;
  }
  return Math.min(1800, Math.max(1, parsed));
};

export default AdminAISettings;
