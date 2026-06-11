import React, { useEffect, useMemo, useState } from 'react';
import InlineNotice from '../../components/InlineNotice';
import { getAISettings, updateAISettings } from '../../api/aiSettings';
import { getErrorMessage } from '../../utils/error';
import { usePreferenceStore } from '../../store/preferences';

const AdminAISettings: React.FC = () => {
  const language = usePreferenceStore((state) => state.language);
  const text = aiSettingsLabels(language);
  const [enabled, setEnabled] = useState(false);
  const [baseUrl, setBaseUrl] = useState('');
  const [model, setModel] = useState('');
  const [apiKey, setApiKey] = useState('');
  const [apiKeyConfigured, setApiKeyConfigured] = useState(false);
  const [clearApiKey, setClearApiKey] = useState(false);
  const [firstByteTimeoutSeconds, setFirstByteTimeoutSeconds] = useState(60);
  const [streamTimeoutSeconds, setStreamTimeoutSeconds] = useState(300);
  const [nonStreamTimeoutSeconds, setNonStreamTimeoutSeconds] = useState(600);
  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState('');
  const [notice, setNotice] = useState('');

  const timeoutInvalid = useMemo(() => (
    firstByteTimeoutSeconds <= 0 ||
    streamTimeoutSeconds < firstByteTimeoutSeconds ||
    nonStreamTimeoutSeconds < firstByteTimeoutSeconds
  ), [firstByteTimeoutSeconds, streamTimeoutSeconds, nonStreamTimeoutSeconds]);

  useEffect(() => {
    let mounted = true;
    setLoading(true);
    setError('');
    getAISettings()
      .then((res) => {
        if (!mounted) {
          return;
        }
        const data = res.data;
        setEnabled(Boolean(data.enabled));
        setBaseUrl(data.baseUrl || '');
        setModel(data.model || '');
        setApiKeyConfigured(Boolean(data.apiKeyConfigured));
        setFirstByteTimeoutSeconds(data.firstByteTimeoutSeconds || 60);
        setStreamTimeoutSeconds(data.streamTimeoutSeconds || 300);
        setNonStreamTimeoutSeconds(data.nonStreamTimeoutSeconds || 600);
      })
      .catch((e) => mounted && setError(getErrorMessage(e, text.loadError)))
      .finally(() => mounted && setLoading(false));
    return () => {
      mounted = false;
    };
  }, [text.loadError]);

  const handleSubmit = async (event: React.FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    setError('');
    setNotice('');
    if (timeoutInvalid) {
      setError(text.timeoutError);
      return;
    }
    setSaving(true);
    try {
      const res = await updateAISettings({
        enabled,
        baseUrl: baseUrl.trim(),
        model: model.trim(),
        apiKey: apiKey.trim(),
        clearApiKey,
        firstByteTimeoutSeconds,
        streamTimeoutSeconds,
        nonStreamTimeoutSeconds,
      });
      setApiKey('');
      setClearApiKey(false);
      setApiKeyConfigured(Boolean(res.data.apiKeyConfigured));
      setNotice(text.saved);
    } catch (e) {
      setError(getErrorMessage(e, text.saveError));
    } finally {
      setSaving(false);
    }
  };

  return (
    <div>
      <div className="mb-8 border-b border-mountain-grey pb-4">
        <h3 className="text-2xl font-bold tracking-widest text-ink">{text.title}</h3>
      </div>

      <InlineNotice message={error} className="mb-6" />
      <InlineNotice message={notice} tone="success" className="mb-6" />

      {loading ? (
        <div className="py-16 text-center tracking-widest text-ink-light">{text.loading}</div>
      ) : (
        <form onSubmit={handleSubmit} className="space-y-8">
          <section className="border border-mountain-grey bg-[var(--paper-soft)] p-5">
            <div className="flex flex-col gap-4 md:flex-row md:items-center md:justify-between">
              <div>
                <h4 className="text-base font-bold tracking-widest text-ink">{text.enableTitle}</h4>
                <p className="mt-2 text-sm leading-relaxed text-ink-light opacity-80">{text.enableDesc}</p>
              </div>
              <button
                type="button"
                role="switch"
                aria-checked={enabled}
                disabled={saving}
                onClick={() => setEnabled((value) => !value)}
                className={`min-w-32 border px-4 py-2 text-sm tracking-widest transition-colors disabled:cursor-not-allowed disabled:opacity-50 ${
                  enabled
                    ? 'border-ochre text-ochre hover:bg-ochre hover:text-paper'
                    : 'border-ink-light text-ink-light hover:border-ink hover:text-ink'
                }`}
              >
                {enabled ? text.enabled : text.disabled}
              </button>
            </div>
          </section>

          <section className="border border-mountain-grey bg-[var(--paper-soft)] p-5">
            <div className="mb-5">
              <h4 className="text-base font-bold tracking-widest text-ink">{text.providerTitle}</h4>
              <p className="mt-2 text-sm leading-relaxed text-ink-light opacity-80">{text.providerDesc}</p>
            </div>
            <div className="grid grid-cols-1 gap-5 md:grid-cols-2">
              <label className="block text-sm text-ink-light">
                <span className="mb-2 block tracking-widest">Base URL</span>
                <input
                  value={baseUrl}
                  onChange={(event) => setBaseUrl(event.target.value)}
                  disabled={saving}
                  placeholder="https://api.example.com/v1"
                  className="w-full border border-mountain-grey bg-transparent px-3 py-2 text-ink outline-none focus:border-ochre disabled:cursor-not-allowed disabled:opacity-50"
                />
              </label>
              <label className="block text-sm text-ink-light">
                <span className="mb-2 block tracking-widest">Model</span>
                <input
                  value={model}
                  onChange={(event) => setModel(event.target.value)}
                  disabled={saving}
                  placeholder="gpt-4.1-mini"
                  className="w-full border border-mountain-grey bg-transparent px-3 py-2 text-ink outline-none focus:border-ochre disabled:cursor-not-allowed disabled:opacity-50"
                />
              </label>
              <label className="block text-sm text-ink-light md:col-span-2">
                <span className="mb-2 block tracking-widest">{text.apiKey}</span>
                <input
                  value={apiKey}
                  onChange={(event) => {
                    setApiKey(event.target.value);
                    if (event.target.value.trim()) {
                      setClearApiKey(false);
                    }
                  }}
                  disabled={saving || clearApiKey}
                  type="password"
                  placeholder={apiKeyConfigured ? text.apiKeyConfigured : text.apiKeyPlaceholder}
                  className="w-full border border-mountain-grey bg-transparent px-3 py-2 text-ink outline-none focus:border-ochre disabled:cursor-not-allowed disabled:opacity-50"
                />
                <div className="mt-3 flex flex-wrap items-center gap-3 text-xs text-ink-light">
                  <span>{apiKeyConfigured ? text.configured : text.notConfigured}</span>
                  {apiKeyConfigured && (
                    <label className="inline-flex items-center gap-2">
                      <input
                        type="checkbox"
                        checked={clearApiKey}
                        disabled={saving || Boolean(apiKey.trim())}
                        onChange={(event) => setClearApiKey(event.target.checked)}
                        className="h-4 w-4 accent-ochre"
                      />
                      <span>{text.clearKey}</span>
                    </label>
                  )}
                </div>
              </label>
            </div>
          </section>

          <section className="border border-mountain-grey bg-[var(--paper-soft)] p-5">
            <div className="mb-5">
              <h4 className="text-base font-bold tracking-widest text-ink">{text.timeoutTitle}</h4>
              <p className="mt-2 text-sm leading-relaxed text-ink-light opacity-80">{text.timeoutDesc}</p>
            </div>
            <div className="grid grid-cols-1 gap-5 md:grid-cols-3">
              <TimeoutInput label={text.firstByte} value={firstByteTimeoutSeconds} disabled={saving} onChange={setFirstByteTimeoutSeconds} />
              <TimeoutInput label={text.stream} value={streamTimeoutSeconds} disabled={saving} onChange={setStreamTimeoutSeconds} />
              <TimeoutInput label={text.nonStream} value={nonStreamTimeoutSeconds} disabled={saving} onChange={setNonStreamTimeoutSeconds} />
            </div>
            {timeoutInvalid && (
              <p className="mt-3 text-sm text-ochre">{text.timeoutError}</p>
            )}
          </section>

          <div className="flex flex-wrap gap-3">
            <button
              type="submit"
              disabled={saving || timeoutInvalid}
              className="border border-ink px-4 py-2 text-sm tracking-widest text-ink transition-colors hover:bg-ink hover:text-paper disabled:cursor-not-allowed disabled:opacity-50"
            >
              {saving ? text.saving : text.save}
            </button>
          </div>
        </form>
      )}
    </div>
  );
};

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

const aiSettingsLabels = (language: string) => language === 'zh'
  ? {
      title: 'AI 配置',
      loading: 'LOADING',
      enableTitle: 'AI 辅助创作',
      enableDesc: '控制后台编辑器中的摘要、SEO 与伴写能力。',
      enabled: '已启用',
      disabled: '已禁用',
      providerTitle: '模型服务',
      providerDesc: '兼容 OpenAI Chat Completions 格式的服务地址、模型和密钥。',
      apiKey: 'API Key',
      apiKeyPlaceholder: '输入新的 API Key',
      apiKeyConfigured: '已保存密钥；留空表示不修改',
      configured: '当前已配置密钥',
      notConfigured: '当前未配置密钥',
      clearKey: '清空已保存密钥',
      timeoutTitle: '超时策略',
      timeoutDesc: '首字等待用于响应头/首字节，流式和非流式分别控制总等待时间。',
      firstByte: '首字等待（秒）',
      stream: '流式输出（秒）',
      nonStream: '非流式输出（秒）',
      timeoutError: '流式和非流式超时不能小于首字等待时间。',
      save: '保存配置',
      saving: '保存中',
      saved: 'AI 配置已保存。',
      loadError: 'AI 配置加载失败',
      saveError: 'AI 配置保存失败',
    }
  : {
      title: 'AI Settings',
      loading: 'LOADING',
      enableTitle: 'AI Writing',
      enableDesc: 'Controls summary, SEO, and assisted writing in the admin editor.',
      enabled: 'Enabled',
      disabled: 'Disabled',
      providerTitle: 'Model Provider',
      providerDesc: 'Service URL, model, and key for OpenAI-compatible Chat Completions.',
      apiKey: 'API Key',
      apiKeyPlaceholder: 'Enter a new API key',
      apiKeyConfigured: 'A key is saved. Leave blank to keep it.',
      configured: 'API key configured',
      notConfigured: 'No API key configured',
      clearKey: 'Clear saved key',
      timeoutTitle: 'Timeouts',
      timeoutDesc: 'First byte controls response header wait; stream and non-stream control total wait.',
      firstByte: 'First byte (s)',
      stream: 'Streaming (s)',
      nonStream: 'Non-streaming (s)',
      timeoutError: 'Stream and non-stream timeouts cannot be lower than first byte timeout.',
      save: 'Save Settings',
      saving: 'Saving',
      saved: 'AI settings saved.',
      loadError: 'Failed to load AI settings',
      saveError: 'Failed to save AI settings',
    };

export default AdminAISettings;
