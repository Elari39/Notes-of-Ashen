import React, { useCallback, useEffect, useMemo, useRef, useState } from 'react';
import InlineNotice from '../../components/InlineNotice';
import PagePendingState from '../../components/RoutePending';
import SettingsCard, { SettingsActions } from '../../components/admin/SettingsCard';
import Button from '../../components/ui/Button';
import Switch from '../../components/ui/Switch';
import { getRAGSettings, getRAGStatus, rebuildRAGIndex, testRAGConnection, updateRAGSettings } from '../../api/rag';
import { useConfirm } from '../../hooks/useConfirm';
import { formatText, translate } from '../../i18n';
import { usePreferenceStore } from '../../store/preferences';
import {
  isValidRAGHistoryRetentionDays,
  RAG_HISTORY_RETENTION_OPTIONS,
  type RAGSettingsResp,
  type RAGStatusResp,
  type RAGTestKind,
  type UpdateRAGSettingsReq,
} from '../../types/api';
import { getErrorMessage } from '../../utils/error';

type RAGSettingsDraft = Omit<RAGSettingsResp, 'apiKeyConfigured' | 'apiKeyNeedsUpdate'> & {
  apiKey: string;
  clearApiKey: boolean;
};

type PendingOperation = 'save' | 'rebuild' | RAGTestKind | null;

const AdminRAGSettings: React.FC = () => {
  const language = usePreferenceStore((state) => state.language);
  const t = useCallback((key: Parameters<typeof translate>[1]) => translate(language, key), [language]);
  const confirm = useConfirm();
  const [draft, setDraft] = useState<RAGSettingsDraft | null>(null);
  const [baseline, setBaseline] = useState<RAGSettingsDraft | null>(null);
  const [status, setStatus] = useState<RAGStatusResp | null>(null);
  const [apiKeyConfigured, setApiKeyConfigured] = useState(false);
  const [apiKeyNeedsUpdate, setApiKeyNeedsUpdate] = useState(false);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');
  const [notice, setNotice] = useState('');
  const [statusError, setStatusError] = useState('');
  const [testError, setTestError] = useState('');
  const [testNotice, setTestNotice] = useState('');
  const [operation, setOperation] = useState<PendingOperation>(null);
  const requestRef = useRef(0);

  const dirty = useMemo(() => draft !== null && baseline !== null && !isSameDraft(draft, baseline), [baseline, draft]);
  const dimensionsInvalid = !draft || !Number.isInteger(draft.embeddingDimensions) || draft.embeddingDimensions < 1 || draft.embeddingDimensions > 32_768;
  const retentionInvalid = !draft || !isValidRAGHistoryRetentionDays(draft.historyRetentionDays);
  const busy = operation !== null;

  const applySettings = useCallback((settings: RAGSettingsResp) => {
    const next = toDraft(settings);
    setDraft(next);
    setBaseline(next);
    setApiKeyConfigured(Boolean(settings.apiKeyConfigured));
    setApiKeyNeedsUpdate(Boolean(settings.apiKeyNeedsUpdate));
  }, []);

  const load = useCallback(async () => {
    const requestId = requestRef.current + 1;
    requestRef.current = requestId;
    setLoading(true);
    setError('');
    setStatusError('');
    try {
      const [settingsResult, statusResult] = await Promise.allSettled([getRAGSettings(), getRAGStatus()]);
      if (requestId !== requestRef.current) return;
      if (settingsResult.status === 'fulfilled') {
        applySettings(settingsResult.value.data);
      } else {
        setDraft(null);
        setBaseline(null);
        setError(getErrorMessage(settingsResult.reason, t('ragSettings.loadError')));
      }
      if (statusResult.status === 'fulfilled') {
        setStatus(statusResult.value.data);
      } else {
        setStatus(null);
        setStatusError(getErrorMessage(statusResult.reason, t('ragSettings.loadError')));
      }
    } finally {
      if (requestId === requestRef.current) setLoading(false);
    }
  }, [applySettings, t]);

  useEffect(() => {
    void load();
    return () => { requestRef.current += 1; };
  }, [load]);

  const updateDraft = (patch: Partial<RAGSettingsDraft>) => {
    setDraft((current) => current ? { ...current, ...patch } : current);
    setError('');
    setNotice('');
    setTestError('');
    setTestNotice('');
  };

  const save = async (event: React.FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    if (!draft || dimensionsInvalid || retentionInvalid || busy) return;
    setOperation('save');
    setError('');
    setNotice('');
    try {
      const response = await updateRAGSettings(toUpdateRequest(draft));
      applySettings(response.data);
      setNotice(t('ragSettings.saved'));
      const statusResponse = await getRAGStatus();
      setStatus(statusResponse.data);
      setStatusError('');
    } catch (requestError) {
      setError(getErrorMessage(requestError, t('ragSettings.saveError')));
    } finally {
      setOperation(null);
    }
  };

  const testConnection = async (kind: RAGTestKind) => {
    if (busy) return;
    setOperation(kind);
    setTestError('');
    setTestNotice('');
    try {
      // 测试已保存配置：API Key 不会在测试请求中回传或记录。
      const response = await testRAGConnection({ kind });
      if (kind === 'embedding' && baseline && response.data.embeddingDimensions !== undefined
        && response.data.embeddingDimensions !== baseline.embeddingDimensions) {
        setTestError(t('ragSettings.dimensionsMismatch'));
      } else {
        setTestNotice(formatText(t('ragSettings.testSuccess'), { latency: response.data.latencyMs }));
      }
    } catch (requestError) {
      setTestError(getErrorMessage(requestError, t('ragSettings.testError')));
    } finally {
      setOperation(null);
    }
  };

  const rebuild = async () => {
    if (busy) return;
    const accepted = await confirm({
      title: t('ragSettings.rebuild'),
      description: t('ragSettings.rebuildConfirm'),
      confirmLabel: t('ragSettings.rebuild'),
      cancelLabel: t('common.cancel'),
      tone: 'danger',
    });
    if (!accepted) return;
    setOperation('rebuild');
    setStatusError('');
    try {
      const response = await rebuildRAGIndex();
      setStatus(response.data);
    } catch (requestError) {
      setStatusError(getErrorMessage(requestError, t('ragSettings.loadError')));
    } finally {
      setOperation(null);
    }
  };

  const restore = () => {
    if (!baseline || busy) return;
    setDraft({ ...baseline });
    setError('');
    setNotice('');
    setTestError('');
    setTestNotice('');
  };

  if (loading) return <PagePendingState variant="admin" label={t('ragSettings.loading')} />;

  if (!draft) {
    return (
      <InlineNotice
        message={error || t('ragSettings.loadError')}
        action={<Button size="sm" onClick={() => void load()}>{t('common.retry')}</Button>}
      />
    );
  }

  return (
    <div>
      <div className="mb-8 border-b border-mountain-grey pb-4">
        <h3 className="text-2xl font-bold tracking-widest text-ink">{t('ragSettings.title')}</h3>
      </div>

      <InlineNotice message={error} className="mb-6" />
      <InlineNotice message={notice} tone="success" icon className="mb-6" />

      <form onSubmit={(event) => void save(event)} className="space-y-8">
        <fieldset disabled={busy} className="space-y-8 disabled:opacity-60">
          <SettingsCard
            title={t('ragSettings.enableTitle')}
            description={t('ragSettings.enableDesc')}
            action={(
              <div className="flex items-center gap-3">
                <Switch checked={draft.enabled} onCheckedChange={(enabled) => updateDraft({ enabled })} label={t('ragSettings.enableTitle')} />
                <span className="text-xs tracking-widest text-ink-light">{draft.enabled ? t('settings.enabled') : t('settings.disabled')}</span>
              </div>
            )}
          />

          <SettingsCard title={t('ragSettings.providerTitle')} description={t('ragSettings.providerDesc')}>
            <div className="grid gap-5 md:grid-cols-2">
              <TextInput label={t('ragSettings.chatBaseUrl')} value={draft.chatBaseUrl} onChange={(chatBaseUrl) => updateDraft({ chatBaseUrl })} />
              <TextInput label={t('ragSettings.embeddingBaseUrl')} value={draft.embeddingBaseUrl} onChange={(embeddingBaseUrl) => updateDraft({ embeddingBaseUrl })} />
              <TextInput label={t('ragSettings.rerankUrl')} value={draft.rerankUrl} onChange={(rerankUrl) => updateDraft({ rerankUrl })} className="md:col-span-2" />
              <TextInput label={t('ragSettings.chatModel')} value={draft.chatModel} onChange={(chatModel) => updateDraft({ chatModel })} />
              <TextInput label={t('ragSettings.embeddingModel')} value={draft.embeddingModel} onChange={(embeddingModel) => updateDraft({ embeddingModel })} />
              <NumberInput label={t('ragSettings.embeddingDimensions')} value={draft.embeddingDimensions} min={1} max={32768} onChange={(embeddingDimensions) => updateDraft({ embeddingDimensions })} invalid={dimensionsInvalid} />
              <TextInput label={t('ragSettings.rerankModel')} value={draft.rerankModel} onChange={(rerankModel) => updateDraft({ rerankModel })} />
              <div className="md:col-span-2">
                <label className="block text-sm text-ink-light">
                  <span className="mb-2 block tracking-widest">{t('ragSettings.apiKey')}</span>
                  <input
                    type="password"
                    autoComplete="new-password"
                    disabled={draft.clearApiKey}
                    value={draft.apiKey}
                    placeholder={apiKeyConfigured ? t('ragSettings.apiKeyConfigured') : t('ragSettings.apiKeyPlaceholder')}
                    onChange={(event) => updateDraft({ apiKey: event.target.value, clearApiKey: false })}
                    className="w-full border border-mountain-grey bg-transparent px-3 py-2 text-ink outline-hidden focus:border-ochre disabled:cursor-not-allowed disabled:opacity-50"
                  />
                </label>
                <div className="mt-3 flex flex-wrap items-center gap-3 text-xs text-ink-light">
                  <span>{apiKeyConfigured ? t('ragSettings.configured') : t('ragSettings.notConfigured')}</span>
                  {apiKeyConfigured && (
                    <label className="inline-flex items-center gap-2">
                      <input
                        type="checkbox"
                        checked={draft.clearApiKey}
                        disabled={Boolean(draft.apiKey.trim())}
                        onChange={(event) => updateDraft({ clearApiKey: event.target.checked })}
                        className="h-4 w-4 accent-ochre"
                      />
                      <span>{t('ragSettings.clearKey')}</span>
                    </label>
                  )}
                </div>
                {apiKeyNeedsUpdate && <InlineNotice message={t('ragSettings.keyNeedsUpdate')} tone="warning" icon className="mt-4" />}
              </div>
            </div>

            <SettingsActions className="mt-5 border-t border-mountain-grey pt-5">
              <Button type="button" variant="ghost" onClick={() => void testConnection('chat')} loading={operation === 'chat'}>{t('ragSettings.testChat')}</Button>
              <Button type="button" variant="ghost" onClick={() => void testConnection('embedding')} loading={operation === 'embedding'}>{t('ragSettings.testEmbedding')}</Button>
              <Button type="button" variant="ghost" onClick={() => void testConnection('rerank')} loading={operation === 'rerank'}>{t('ragSettings.testRerank')}</Button>
            </SettingsActions>
            <p className="mt-3 text-xs leading-relaxed text-ink-light">{t('ragSettings.testSavedHint')}</p>
            <InlineNotice message={testError} className="mt-4" />
            <InlineNotice message={testNotice} tone="success" icon className="mt-4" />
          </SettingsCard>

          <SettingsCard title={t('ragSettings.retention')} description={t('ragSettings.retentionHint')}>
            <label className="block max-w-md text-sm text-ink-light">
              <span className="mb-2 block tracking-widest">{t('ragSettings.retention')}</span>
              <select
                value={draft.historyRetentionDays}
                onChange={(event) => updateDraft({ historyRetentionDays: Number(event.target.value) })}
                className="w-full border border-mountain-grey bg-transparent px-3 py-2 text-ink outline-hidden focus:border-ochre"
              >
                {RAG_HISTORY_RETENTION_OPTIONS.map((days) => (
                  <option key={days} value={days}>{retentionOptionLabel(days, t)}</option>
                ))}
              </select>
            </label>
            {retentionInvalid && <InlineNotice message={t('ragSettings.retentionInvalid')} className="mt-4" />}
          </SettingsCard>

          <SettingsCard title={t('ragSettings.statusTitle')} description={t('ragSettings.statusDesc')}>
            <InlineNotice message={statusError} className="mb-4" />
            {status ? <RAGStatusCard status={status} t={t} /> : <p className="text-sm text-ink-light">—</p>}
            <SettingsActions className="mt-5 border-t border-mountain-grey pt-5">
              <Button type="button" variant="ghost" onClick={() => void load()}>{t('common.retry')}</Button>
              <Button type="button" variant="danger" onClick={() => void rebuild()} loading={operation === 'rebuild'}>{t('ragSettings.rebuild')}</Button>
            </SettingsActions>
          </SettingsCard>

          <SettingsActions>
            <Button type="submit" variant="primary" disabled={!dirty || dimensionsInvalid || retentionInvalid} loading={operation === 'save'}>
              {operation === 'save' ? t('ragSettings.saving') : t('ragSettings.save')}
            </Button>
            <Button type="button" variant="ghost" disabled={!dirty} onClick={restore}>{t('common.cancel')}</Button>
          </SettingsActions>
        </fieldset>
      </form>
    </div>
  );
};

const TextInput = ({ label, value, onChange, className = '' }: { label: string; value: string; onChange: (value: string) => void; className?: string }) => (
  <label className={`block text-sm text-ink-light ${className}`.trim()}>
    <span className="mb-2 block tracking-widest">{label}</span>
    <input value={value} onChange={(event) => onChange(event.target.value)} className="w-full border border-mountain-grey bg-transparent px-3 py-2 text-ink outline-hidden focus:border-ochre" />
  </label>
);

const NumberInput = ({ label, value, min, max, onChange, invalid }: { label: string; value: number; min: number; max: number; onChange: (value: number) => void; invalid: boolean }) => (
  <label className="block text-sm text-ink-light">
    <span className="mb-2 block tracking-widest">{label}</span>
    <input
      type="number"
      min={min}
      max={max}
      value={Number.isFinite(value) ? value : ''}
      onChange={(event) => onChange(Number(event.target.value))}
      className={`w-full border bg-transparent px-3 py-2 text-ink outline-hidden focus:border-ochre ${invalid ? 'border-ember' : 'border-mountain-grey'}`}
    />
  </label>
);

const RAGStatusCard = ({ status, t }: { status: RAGStatusResp; t: (key: Parameters<typeof translate>[1]) => string }) => (
  <div className="grid gap-3 sm:grid-cols-2 lg:grid-cols-4">
    <StatusMetric label={t(`ragSettings.status.${status.status}` as Parameters<typeof translate>[1])} value={status.status} />
    {status.queueDepth !== undefined && <StatusMetric label={t('ragSettings.queueDepth')} value={formatText(t('ragSettings.queueDepth'), { count: status.queueDepth })} />}
    {status.indexedArticles !== undefined && <StatusMetric label={t('ragSettings.indexedArticles')} value={formatText(t('ragSettings.indexedArticles'), { count: status.indexedArticles })} />}
    {status.indexedChunks !== undefined && <StatusMetric label={t('ragSettings.indexedChunks')} value={formatText(t('ragSettings.indexedChunks'), { count: status.indexedChunks })} />}
    {status.lastError && <InlineNotice message={status.lastError} className="sm:col-span-2 lg:col-span-4" />}
  </div>
);

const StatusMetric = ({ label, value }: { label: string; value: string }) => (
  <div className="rounded-md border border-hairline bg-surface-soft px-3 py-3">
    <p className="text-[0.65rem] tracking-widest text-muted">{label}</p>
    <p className="mt-1 truncate text-sm font-medium text-ink">{value}</p>
  </div>
);

const toDraft = (settings: RAGSettingsResp): RAGSettingsDraft => ({
  enabled: Boolean(settings.enabled),
  chatBaseUrl: settings.chatBaseUrl || '',
  embeddingBaseUrl: settings.embeddingBaseUrl || '',
  rerankUrl: settings.rerankUrl || '',
  chatModel: settings.chatModel || '',
  embeddingModel: settings.embeddingModel || '',
  embeddingDimensions: settings.embeddingDimensions || 1024,
  rerankModel: settings.rerankModel || '',
  historyRetentionDays: isValidRAGHistoryRetentionDays(settings.historyRetentionDays) ? settings.historyRetentionDays : 90,
  apiKey: '',
  clearApiKey: false,
});

const toUpdateRequest = (draft: RAGSettingsDraft): UpdateRAGSettingsReq => ({
  enabled: draft.enabled,
  chatBaseUrl: draft.chatBaseUrl.trim(),
  embeddingBaseUrl: draft.embeddingBaseUrl.trim(),
  rerankUrl: draft.rerankUrl.trim(),
  chatModel: draft.chatModel.trim(),
  embeddingModel: draft.embeddingModel.trim(),
  embeddingDimensions: draft.embeddingDimensions,
  rerankModel: draft.rerankModel.trim(),
  historyRetentionDays: draft.historyRetentionDays,
  ...(draft.apiKey.trim() ? { apiKey: draft.apiKey.trim() } : {}),
  ...(draft.clearApiKey ? { clearApiKey: true } : {}),
});

const isSameDraft = (left: RAGSettingsDraft, right: RAGSettingsDraft): boolean => (
  left.enabled === right.enabled
  && left.chatBaseUrl === right.chatBaseUrl
  && left.embeddingBaseUrl === right.embeddingBaseUrl
  && left.rerankUrl === right.rerankUrl
  && left.chatModel === right.chatModel
  && left.embeddingModel === right.embeddingModel
  && left.embeddingDimensions === right.embeddingDimensions
  && left.rerankModel === right.rerankModel
  && left.historyRetentionDays === right.historyRetentionDays
  && left.apiKey === right.apiKey
  && left.clearApiKey === right.clearApiKey
);

const retentionOptionLabel = (
  days: typeof RAG_HISTORY_RETENTION_OPTIONS[number],
  t: (key: Parameters<typeof translate>[1]) => string,
): string => (days === 0 ? t('ragSettings.retentionForever') : formatText(t('ragSettings.retentionDays'), { days }));

export default AdminRAGSettings;
