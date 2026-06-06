import React, { useEffect, useState } from 'react';
import InlineNotice from '../../components/InlineNotice';
import { getErrorMessage } from '../../utils/error';
import { translate } from '../../i18n';
import { usePreferenceStore } from '../../store/preferences';
import { useSiteSettingsStore } from '../../store/siteSettings';
import type { HomeArticleLayout } from '../../types';

const AdminSettings: React.FC = () => {
  const language = usePreferenceStore((state) => state.language);
  const {
    registrationEnabled,
    homeArticleLayout,
    isLoading,
    fetchSettings,
    updateSettings,
  } = useSiteSettingsStore();
  const [draftRegistrationEnabled, setDraftRegistrationEnabled] = useState(registrationEnabled);
  const [draftHomeArticleLayout, setDraftHomeArticleLayout] = useState<HomeArticleLayout>(homeArticleLayout);
  const [error, setError] = useState('');
  const [notice, setNotice] = useState('');
  const t = (key: Parameters<typeof translate>[1]) => translate(language, key);

  useEffect(() => {
    fetchSettings();
  }, [fetchSettings]);

  useEffect(() => {
    setDraftRegistrationEnabled(registrationEnabled);
    setDraftHomeArticleLayout(homeArticleLayout);
  }, [registrationEnabled, homeArticleLayout]);

  const hasChanges = draftRegistrationEnabled !== registrationEnabled || draftHomeArticleLayout !== homeArticleLayout;

  const handleSubmit = async (event: React.FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    setError('');
    setNotice('');
    try {
      await updateSettings({
        registrationEnabled: draftRegistrationEnabled,
        homeArticleLayout: draftHomeArticleLayout,
      });
      setNotice(t('settings.saved'));
    } catch (e: unknown) {
      setError(getErrorMessage(e, t('settings.saveError')));
    }
  };

  const handleReset = () => {
    setDraftRegistrationEnabled(registrationEnabled);
    setDraftHomeArticleLayout(homeArticleLayout);
    setError('');
    setNotice('');
  };

  return (
    <div>
      <div className="mb-8 border-b border-mountain-grey pb-4">
        <h3 className="text-2xl font-bold text-ink tracking-widest">{t('admin.settings')}</h3>
      </div>

      <InlineNotice message={error} className="mb-6" />
      <InlineNotice message={notice} tone="success" className="mb-6" />

      <form onSubmit={handleSubmit} className="space-y-8">
        <section className="border border-mountain-grey bg-[var(--paper-soft)] p-5">
          <div className="flex flex-col gap-4 md:flex-row md:items-center md:justify-between">
            <div>
              <h4 className="text-base font-bold tracking-widest text-ink">{t('settings.registrationTitle')}</h4>
              <p className="mt-2 text-sm leading-relaxed text-ink-light opacity-80">
                {t('settings.registrationDesc')}
              </p>
            </div>
            <button
              type="button"
              role="switch"
              aria-checked={draftRegistrationEnabled}
              disabled={isLoading}
              onClick={() => setDraftRegistrationEnabled((enabled) => !enabled)}
              className={`min-w-32 border px-4 py-2 text-sm tracking-widest transition-colors disabled:cursor-not-allowed disabled:opacity-50 ${
                draftRegistrationEnabled
                  ? 'border-ochre text-ochre hover:bg-ochre hover:text-paper'
                  : 'border-ink-light text-ink-light hover:border-ink hover:text-ink'
              }`}
            >
              {draftRegistrationEnabled ? t('settings.registrationEnabled') : t('settings.registrationDisabled')}
            </button>
          </div>
        </section>

        <section className="border border-mountain-grey bg-[var(--paper-soft)] p-5">
          <div className="mb-5">
            <h4 className="text-base font-bold tracking-widest text-ink">{t('settings.homeLayoutTitle')}</h4>
            <p className="mt-2 text-sm leading-relaxed text-ink-light opacity-80">
              {t('settings.homeLayoutDesc')}
            </p>
          </div>
          <div className="grid grid-cols-1 border border-mountain-grey md:grid-cols-2">
            <button
              type="button"
              disabled={isLoading}
              onClick={() => setDraftHomeArticleLayout('standard')}
              className={`px-4 py-4 text-left transition-colors disabled:cursor-not-allowed disabled:opacity-50 ${
                draftHomeArticleLayout === 'standard'
                  ? 'bg-ink text-paper'
                  : 'text-ink-light hover:text-ochre'
              }`}
            >
              <span className="block text-sm font-bold tracking-widest">{t('settings.homeLayoutStandard')}</span>
              <span className="mt-2 block text-xs leading-relaxed opacity-75">{t('settings.homeLayoutStandardDesc')}</span>
            </button>
            <button
              type="button"
              disabled={isLoading}
              onClick={() => setDraftHomeArticleLayout('alternating')}
              className={`border-t border-mountain-grey px-4 py-4 text-left transition-colors disabled:cursor-not-allowed disabled:opacity-50 md:border-l md:border-t-0 ${
                draftHomeArticleLayout === 'alternating'
                  ? 'bg-ink text-paper'
                  : 'text-ink-light hover:text-ochre'
              }`}
            >
              <span className="block text-sm font-bold tracking-widest">{t('settings.homeLayoutAlternating')}</span>
              <span className="mt-2 block text-xs leading-relaxed opacity-75">{t('settings.homeLayoutAlternatingDesc')}</span>
            </button>
          </div>
        </section>

        <div className="flex flex-wrap gap-3">
          <button
            type="submit"
            disabled={isLoading || !hasChanges}
            className="px-4 py-2 border border-ink text-ink hover:bg-ink hover:text-paper tracking-widest text-sm transition-colors disabled:cursor-not-allowed disabled:opacity-50"
          >
            {isLoading ? t('common.saving') : t('common.save')}
          </button>
          <button
            type="button"
            disabled={isLoading || !hasChanges}
            onClick={handleReset}
            className="px-4 py-2 border border-mountain-grey text-ink-light hover:border-ochre hover:text-ochre tracking-widest text-sm transition-colors disabled:cursor-not-allowed disabled:opacity-50"
          >
            {t('common.cancel')}
          </button>
        </div>
      </form>
    </div>
  );
};

export default AdminSettings;
