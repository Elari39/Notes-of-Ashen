import React, { useEffect, useMemo, useRef, useState } from 'react';
import InlineNotice from '../../components/InlineNotice';
import PagePendingState from '../../components/RoutePending';
import Switch from '../../components/ui/Switch';
import Button from '../../components/ui/Button';
import { getAISettings, updateAISettings } from '../../api/aiSettings';
import { getErrorMessage } from '../../utils/error';
import { usePreferenceStore } from '../../store/preferences';
import { translate } from '../../i18n';

const AdminAISettings: React.FC = () => {
  const language = usePreferenceStore((state) => state.language);
  const t = (key: Parameters<typeof translate>[1]) => translate(language, key);
  // 用 ref 持有最新 language，避免切换语言时重新拉取 AI 设置（数据与语言无关）。
  const languageRef = useRef(language);
  languageRef.current = language;
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
      .catch((e) => mounted && setError(getErrorMessage(e, translate(languageRef.current, 'aiSettings.loadError'))))
      .finally(() => mounted && setLoading(false));
    return () => {
      mounted = false;
    };
  }, []);

  const handleSubmit = async (event: React.FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    setError('');
    setNotice('');
    if (timeoutInvalid) {
      setError(t('aiSettings.timeoutError'));
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
      setNotice(t('aiSettings.saved'));
    } catch (e) {
      setError(getErrorMessage(e, t('aiSettings.saveError')));
    } finally {
      setSaving(false);
    }
  };

  return (
    <div>
      <div className="mb-8 border-b border-mountain-grey pb-4">
        <h3 className="text-2xl font-bold tracking-widest text-ink">{t('aiSettings.title')}</h3>
      </div>

      <InlineNotice message={error} className="mb-6" />
      <InlineNotice message={notice} tone="success" className="mb-6" />

      {loading && <PagePendingState variant="admin" label={t('aiSettings.loading')} />}
      {!loading && (
        <form onSubmit={handleSubmit} className="space-y-8">
          <section className="border border-mountain-grey bg-[var(--paper-soft)] p-5">
            <div className="flex flex-col gap-4 md:flex-row md:items-center md:justify-between">
              <div>
                <h4 className="text-base font-bold tracking-widest text-ink">{t('aiSettings.enableTitle')}</h4>
                <p className="mt-2 text-sm leading-relaxed text-ink-light opacity-80">{t('aiSettings.enableDesc')}</p>
              </div>
              <div className="flex items-center gap-3">
                <Switch
                  checked={enabled}
                  onCheckedChange={setEnabled}
                  disabled={saving}
                  label={t('aiSettings.enableTitle')}
                />
                <span className="text-xs tracking-widest text-ink-light">
                  {enabled ? t('settings.enabled') : t('settings.disabled')}
                </span>
              </div>
            </div>
          </section>

          <section className="border border-mountain-grey bg-[var(--paper-soft)] p-5">
            <div className="mb-5">
              <h4 className="text-base font-bold tracking-widest text-ink">{t('aiSettings.providerTitle')}</h4>
              <p className="mt-2 text-sm leading-relaxed text-ink-light opacity-80">{t('aiSettings.providerDesc')}</p>
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
                <span className="mb-2 block tracking-widest">{t('aiSettings.apiKey')}</span>
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
                  placeholder={apiKeyConfigured ? t('aiSettings.apiKeyConfigured') : t('aiSettings.apiKeyPlaceholder')}
                  className="w-full border border-mountain-grey bg-transparent px-3 py-2 text-ink outline-none focus:border-ochre disabled:cursor-not-allowed disabled:opacity-50"
                />
                <div className="mt-3 flex flex-wrap items-center gap-3 text-xs text-ink-light">
                  <span>{apiKeyConfigured ? t('aiSettings.configured') : t('aiSettings.notConfigured')}</span>
                  {apiKeyConfigured && (
                    <label className="inline-flex items-center gap-2">
                      <input
                        type="checkbox"
                        checked={clearApiKey}
                        disabled={saving || Boolean(apiKey.trim())}
                        onChange={(event) => setClearApiKey(event.target.checked)}
                        className="h-4 w-4 accent-ochre"
                      />
                      <span>{t('aiSettings.clearKey')}</span>
                    </label>
                  )}
                </div>
              </label>
            </div>
          </section>

          <section className="border border-mountain-grey bg-[var(--paper-soft)] p-5">
            <div className="mb-5">
              <h4 className="text-base font-bold tracking-widest text-ink">{t('aiSettings.timeoutTitle')}</h4>
              <p className="mt-2 text-sm leading-relaxed text-ink-light opacity-80">{t('aiSettings.timeoutDesc')}</p>
            </div>
            <div className="grid grid-cols-1 gap-5 md:grid-cols-3">
              <TimeoutInput label={t('aiSettings.firstByte')} value={firstByteTimeoutSeconds} disabled={saving} onChange={setFirstByteTimeoutSeconds} />
              <TimeoutInput label={t('aiSettings.stream')} value={streamTimeoutSeconds} disabled={saving} onChange={setStreamTimeoutSeconds} />
              <TimeoutInput label={t('aiSettings.nonStream')} value={nonStreamTimeoutSeconds} disabled={saving} onChange={setNonStreamTimeoutSeconds} />
            </div>
            {timeoutInvalid && (
              <p className="mt-3 text-sm text-ochre">{t('aiSettings.timeoutError')}</p>
            )}
          </section>

          <div className="flex flex-wrap gap-3">
            <Button
              type="submit"
              variant="primary"
              size="md"
              disabled={timeoutInvalid}
              loading={saving}
            >
              {saving ? t('aiSettings.saving') : t('aiSettings.save')}
            </Button>
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

export default AdminAISettings;
