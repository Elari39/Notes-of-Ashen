import React, { useEffect, useState } from 'react';
import InlineNotice from '../../components/InlineNotice';
import PagePendingState from '../../components/RoutePending';
import Switch from '../../components/ui/Switch';
import Button from '../../components/ui/Button';
import SettingsCard, { SettingsActions } from '../../components/admin/SettingsCard';
import { getErrorMessage } from '../../utils/error';
import { translate } from '../../i18n';
import { usePreferenceStore } from '../../store/preferences';
import { useSiteSettingsStore } from '../../store/siteSettings';
import { areSiteSettingsControlsDisabled } from '../../store/siteSettingsPolicy';
import type { HomeArticleLayout } from '../../types';
import { useShallow } from 'zustand/react/shallow';

const AdminSettings: React.FC = () => {
  const language = usePreferenceStore((state) => state.language);
  const {
    registrationEnabled,
    homeArticleLayout,
    siteTitle,
    siteDescription,
    siteKeywords,
    siteBaseUrl,
    resumePageEnabled,
    resumeNavHidden,
    projectsPageEnabled,
    projectsNavHidden,
    isLoading,
    hasLoaded,
    loadError,
    fetchSettings,
    updateSettings,
  } = useSiteSettingsStore(
    useShallow((state) => ({
      registrationEnabled: state.registrationEnabled,
      homeArticleLayout: state.homeArticleLayout,
      siteTitle: state.siteTitle,
      siteDescription: state.siteDescription,
      siteKeywords: state.siteKeywords,
      siteBaseUrl: state.siteBaseUrl,
      resumePageEnabled: state.resumePageEnabled,
      resumeNavHidden: state.resumeNavHidden,
      projectsPageEnabled: state.projectsPageEnabled,
      projectsNavHidden: state.projectsNavHidden,
      isLoading: state.isLoading,
      hasLoaded: state.hasLoaded,
      loadError: state.loadError,
      fetchSettings: state.fetchSettings,
      updateSettings: state.updateSettings,
    })),
  );
  const [draftRegistrationEnabled, setDraftRegistrationEnabled] = useState(registrationEnabled);
  const [draftHomeArticleLayout, setDraftHomeArticleLayout] = useState<HomeArticleLayout>(homeArticleLayout);
  const [draftSiteTitle, setDraftSiteTitle] = useState(siteTitle);
  const [draftSiteDescription, setDraftSiteDescription] = useState(siteDescription);
  const [draftSiteKeywords, setDraftSiteKeywords] = useState(siteKeywords);
  const [draftSiteBaseUrl, setDraftSiteBaseUrl] = useState(siteBaseUrl);
  const [draftResumePageEnabled, setDraftResumePageEnabled] = useState(resumePageEnabled);
  const [draftResumeNavHidden, setDraftResumeNavHidden] = useState(resumeNavHidden);
  const [draftProjectsPageEnabled, setDraftProjectsPageEnabled] = useState(projectsPageEnabled);
  const [draftProjectsNavHidden, setDraftProjectsNavHidden] = useState(projectsNavHidden);
  const [error, setError] = useState('');
  const [notice, setNotice] = useState('');
  const t = (key: Parameters<typeof translate>[1]) => translate(language, key);
  const controlsDisabled = areSiteSettingsControlsDisabled(hasLoaded, isLoading);

  useEffect(() => {
    fetchSettings();
  }, [fetchSettings]);

  useEffect(() => {
    setDraftRegistrationEnabled(registrationEnabled);
    setDraftHomeArticleLayout(homeArticleLayout);
    setDraftSiteTitle(siteTitle);
    setDraftSiteDescription(siteDescription);
    setDraftSiteKeywords(siteKeywords);
    setDraftSiteBaseUrl(siteBaseUrl);
    setDraftResumePageEnabled(resumePageEnabled);
    setDraftResumeNavHidden(resumeNavHidden);
    setDraftProjectsPageEnabled(projectsPageEnabled);
    setDraftProjectsNavHidden(projectsNavHidden);
  }, [registrationEnabled, homeArticleLayout, siteTitle, siteDescription, siteKeywords, siteBaseUrl, resumePageEnabled, resumeNavHidden, projectsPageEnabled, projectsNavHidden]);

  const hasChanges =
    draftRegistrationEnabled !== registrationEnabled ||
    draftHomeArticleLayout !== homeArticleLayout ||
    draftSiteTitle !== siteTitle ||
    draftSiteDescription !== siteDescription ||
    draftSiteKeywords !== siteKeywords ||
    draftSiteBaseUrl !== siteBaseUrl ||
    draftResumePageEnabled !== resumePageEnabled ||
    draftResumeNavHidden !== resumeNavHidden ||
    draftProjectsPageEnabled !== projectsPageEnabled ||
    draftProjectsNavHidden !== projectsNavHidden;

  const handleSubmit = async (event: React.FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    setError('');
    setNotice('');
    try {
      await updateSettings({
        registrationEnabled: draftRegistrationEnabled,
        homeArticleLayout: draftHomeArticleLayout,
        siteTitle: draftSiteTitle.trim(),
        siteDescription: draftSiteDescription.trim(),
        siteKeywords: draftSiteKeywords.trim(),
        siteBaseUrl: draftSiteBaseUrl.trim(),
        resumePageEnabled: draftResumePageEnabled,
        resumeNavHidden: draftResumeNavHidden,
        projectsPageEnabled: draftProjectsPageEnabled,
        projectsNavHidden: draftProjectsNavHidden,
      });
      setNotice(t('settings.saved'));
    } catch (e: unknown) {
      setError(getErrorMessage(e, t('settings.saveError')));
    }
  };

  const handleReset = () => {
    setDraftRegistrationEnabled(registrationEnabled);
    setDraftHomeArticleLayout(homeArticleLayout);
    setDraftSiteTitle(siteTitle);
    setDraftSiteDescription(siteDescription);
    setDraftSiteKeywords(siteKeywords);
    setDraftSiteBaseUrl(siteBaseUrl);
    setDraftResumePageEnabled(resumePageEnabled);
    setDraftResumeNavHidden(resumeNavHidden);
    setDraftProjectsPageEnabled(projectsPageEnabled);
    setDraftProjectsNavHidden(projectsNavHidden);
    setError('');
    setNotice('');
  };

  return (
    <div>
      <div className="mb-8 border-b border-mountain-grey pb-4">
        <h3 className="text-2xl font-bold tracking-widest text-ink">{t('admin.settings')}</h3>
      </div>

      {hasLoaded && <InlineNotice message={error} className="mb-6" />}
      {hasLoaded && <InlineNotice message={notice} tone="success" className="mb-6" />}
      {!hasLoaded && !loadError && <PagePendingState variant="admin" label={t('common.loading')} />}
      {!hasLoaded && loadError && (
        <InlineNotice
          message={t('siteSettings.loadError')}
          className="mb-6"
          action={(
            <Button size="sm" onClick={() => void fetchSettings()}>
              {t('common.retry')}
            </Button>
          )}
        />
      )}

      {hasLoaded && (
        <form onSubmit={handleSubmit} className="space-y-8">
          <fieldset disabled={controlsDisabled} className="space-y-8 disabled:opacity-60">
            <SettingsCard
              title={t('settings.registrationTitle')}
              description={t('settings.registrationDesc')}
              action={(
                <div className="flex items-center gap-3">
                  <Switch
                    checked={draftRegistrationEnabled}
                    onCheckedChange={setDraftRegistrationEnabled}
                    disabled={isLoading}
                    label={t('settings.registrationTitle')}
                  />
                  <span className="text-xs tracking-widest text-ink-light">
                    {draftRegistrationEnabled ? t('settings.registrationEnabled') : t('settings.registrationDisabled')}
                  </span>
                </div>
              )}
            />

            <SettingsCard
              title={t('settings.publicPagesTitle')}
              description={t('settings.publicPagesDesc')}
            >
              <div className="grid grid-cols-1 gap-4 md:grid-cols-2">
                <div className="border border-mountain-grey p-4">
                  <div className="mb-4">
                    <p className="text-sm font-bold tracking-widest text-ink">{t('settings.resumePage')}</p>
                    <p className="mt-2 text-xs leading-relaxed text-ink-light opacity-80">{t('settings.resumePageDesc')}</p>
                  </div>
                  <div className="flex flex-col gap-3">
                    <div className="flex items-center gap-3">
                      <Switch
                        checked={draftResumePageEnabled}
                        onCheckedChange={setDraftResumePageEnabled}
                        disabled={isLoading}
                        label={t('settings.resumePage')}
                      />
                      <span className="text-xs tracking-widest text-ink-light">
                        {draftResumePageEnabled ? t('settings.enabled') : t('settings.disabled')}
                      </span>
                    </div>
                    <div className="flex items-center gap-3">
                      <Switch
                        checked={!draftResumeNavHidden}
                        onCheckedChange={(next) => setDraftResumeNavHidden(!next)}
                        disabled={isLoading}
                        label={t('settings.navVisible')}
                      />
                      <span className="text-xs tracking-widest text-ink-light">
                        {draftResumeNavHidden ? t('settings.navHidden') : t('settings.navVisible')}
                      </span>
                    </div>
                  </div>
                </div>

                <div className="border border-mountain-grey p-4">
                  <div className="mb-4">
                    <p className="text-sm font-bold tracking-widest text-ink">{t('settings.projectsPage')}</p>
                    <p className="mt-2 text-xs leading-relaxed text-ink-light opacity-80">{t('settings.projectsPageDesc')}</p>
                  </div>
                  <div className="flex flex-col gap-3">
                    <div className="flex items-center gap-3">
                      <Switch
                        checked={draftProjectsPageEnabled}
                        onCheckedChange={setDraftProjectsPageEnabled}
                        disabled={isLoading}
                        label={t('settings.projectsPage')}
                      />
                      <span className="text-xs tracking-widest text-ink-light">
                        {draftProjectsPageEnabled ? t('settings.enabled') : t('settings.disabled')}
                      </span>
                    </div>
                    <div className="flex items-center gap-3">
                      <Switch
                        checked={!draftProjectsNavHidden}
                        onCheckedChange={(next) => setDraftProjectsNavHidden(!next)}
                        disabled={isLoading}
                        label={t('settings.navVisible')}
                      />
                      <span className="text-xs tracking-widest text-ink-light">
                        {draftProjectsNavHidden ? t('settings.navHidden') : t('settings.navVisible')}
                      </span>
                    </div>
                  </div>
                </div>
              </div>
            </SettingsCard>

            <SettingsCard
              title={t('settings.homeLayoutTitle')}
              description={t('settings.homeLayoutDesc')}
            >
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
            </SettingsCard>

            <SettingsCard
              title={t('settings.seoTitle')}
              description={t('settings.seoDesc')}
            >
              <div className="grid grid-cols-1 gap-5 md:grid-cols-2">
                <label className="block text-sm text-ink-light">
                  <span className="mb-2 block tracking-widest">{t('settings.siteTitle')}</span>
                  <input
                    value={draftSiteTitle}
                    onChange={(event) => setDraftSiteTitle(event.target.value)}
                    disabled={isLoading}
                    className="w-full border border-mountain-grey bg-transparent px-3 py-2 text-ink outline-none focus:border-ochre disabled:cursor-not-allowed disabled:opacity-50"
                  />
                </label>
                <label className="block text-sm text-ink-light">
                  <span className="mb-2 block tracking-widest">{t('settings.siteBaseUrl')}</span>
                  <input
                    value={draftSiteBaseUrl}
                    onChange={(event) => setDraftSiteBaseUrl(event.target.value)}
                    disabled={isLoading}
                    placeholder="https://example.com"
                    className="w-full border border-mountain-grey bg-transparent px-3 py-2 text-ink outline-none focus:border-ochre disabled:cursor-not-allowed disabled:opacity-50"
                  />
                </label>
                <label className="block text-sm text-ink-light md:col-span-2">
                  <span className="mb-2 block tracking-widest">{t('settings.siteDescription')}</span>
                  <textarea
                    value={draftSiteDescription}
                    onChange={(event) => setDraftSiteDescription(event.target.value)}
                    disabled={isLoading}
                    rows={3}
                    className="w-full resize-none border border-mountain-grey bg-transparent px-3 py-2 text-ink outline-none focus:border-ochre disabled:cursor-not-allowed disabled:opacity-50"
                  />
                </label>
                <label className="block text-sm text-ink-light md:col-span-2">
                  <span className="mb-2 block tracking-widest">{t('settings.siteKeywords')}</span>
                  <input
                    value={draftSiteKeywords}
                    onChange={(event) => setDraftSiteKeywords(event.target.value)}
                    disabled={isLoading}
                    placeholder="blog,notes,writing"
                    className="w-full border border-mountain-grey bg-transparent px-3 py-2 text-ink outline-none focus:border-ochre disabled:cursor-not-allowed disabled:opacity-50"
                  />
                </label>
              </div>
            </SettingsCard>

            <SettingsActions>
              <Button
                type="submit"
                variant="primary"
                size="md"
                disabled={!hasChanges}
                loading={isLoading}
              >
                {isLoading ? t('common.saving') : t('common.save')}
              </Button>
              <Button
                type="button"
                variant="ghost"
                size="md"
                disabled={isLoading || !hasChanges}
                onClick={handleReset}
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

export default AdminSettings;
