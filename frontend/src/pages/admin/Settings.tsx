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
    siteTitle,
    siteDescription,
    siteKeywords,
    siteBaseUrl,
    resumePageEnabled,
    resumeNavHidden,
    projectsPageEnabled,
    projectsNavHidden,
    isLoading,
    fetchSettings,
    updateSettings,
  } = useSiteSettingsStore();
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
            <h4 className="text-base font-bold tracking-widest text-ink">前台页面控制</h4>
            <p className="mt-2 text-sm leading-relaxed text-ink-light opacity-80">
              启用决定路由是否可访问；隐藏只控制是否出现在前台导航中。
            </p>
          </div>
          <div className="grid grid-cols-1 gap-4 md:grid-cols-2">
            <div className="border border-mountain-grey p-4">
              <div className="mb-4">
                <p className="text-sm font-bold tracking-widest text-ink">简介页面</p>
                <p className="mt-2 text-xs leading-relaxed text-ink-light opacity-80">控制 /resume 页面和导航入口。</p>
              </div>
              <div className="flex flex-wrap gap-3">
                <button
                  type="button"
                  role="switch"
                  aria-checked={draftResumePageEnabled}
                  disabled={isLoading}
                  onClick={() => setDraftResumePageEnabled((enabled) => !enabled)}
                  className={`min-w-28 border px-4 py-2 text-sm tracking-widest transition-colors disabled:cursor-not-allowed disabled:opacity-50 ${
                    draftResumePageEnabled
                      ? 'border-ochre text-ochre hover:bg-ochre hover:text-paper'
                      : 'border-ink-light text-ink-light hover:border-ink hover:text-ink'
                  }`}
                >
                  {draftResumePageEnabled ? '已启用' : '已禁用'}
                </button>
                <button
                  type="button"
                  role="switch"
                  aria-checked={!draftResumeNavHidden}
                  disabled={isLoading}
                  onClick={() => setDraftResumeNavHidden((hidden) => !hidden)}
                  className={`min-w-28 border px-4 py-2 text-sm tracking-widest transition-colors disabled:cursor-not-allowed disabled:opacity-50 ${
                    !draftResumeNavHidden
                      ? 'border-ochre text-ochre hover:bg-ochre hover:text-paper'
                      : 'border-ink-light text-ink-light hover:border-ink hover:text-ink'
                  }`}
                >
                  {draftResumeNavHidden ? '导航隐藏' : '导航显示'}
                </button>
              </div>
            </div>

            <div className="border border-mountain-grey p-4">
              <div className="mb-4">
                <p className="text-sm font-bold tracking-widest text-ink">项目页面</p>
                <p className="mt-2 text-xs leading-relaxed text-ink-light opacity-80">控制 /projects 页面和导航入口。</p>
              </div>
              <div className="flex flex-wrap gap-3">
                <button
                  type="button"
                  role="switch"
                  aria-checked={draftProjectsPageEnabled}
                  disabled={isLoading}
                  onClick={() => setDraftProjectsPageEnabled((enabled) => !enabled)}
                  className={`min-w-28 border px-4 py-2 text-sm tracking-widest transition-colors disabled:cursor-not-allowed disabled:opacity-50 ${
                    draftProjectsPageEnabled
                      ? 'border-ochre text-ochre hover:bg-ochre hover:text-paper'
                      : 'border-ink-light text-ink-light hover:border-ink hover:text-ink'
                  }`}
                >
                  {draftProjectsPageEnabled ? '已启用' : '已禁用'}
                </button>
                <button
                  type="button"
                  role="switch"
                  aria-checked={!draftProjectsNavHidden}
                  disabled={isLoading}
                  onClick={() => setDraftProjectsNavHidden((hidden) => !hidden)}
                  className={`min-w-28 border px-4 py-2 text-sm tracking-widest transition-colors disabled:cursor-not-allowed disabled:opacity-50 ${
                    !draftProjectsNavHidden
                      ? 'border-ochre text-ochre hover:bg-ochre hover:text-paper'
                      : 'border-ink-light text-ink-light hover:border-ink hover:text-ink'
                  }`}
                >
                  {draftProjectsNavHidden ? '导航隐藏' : '导航显示'}
                </button>
              </div>
            </div>
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

        <section className="border border-mountain-grey bg-[var(--paper-soft)] p-5">
          <div className="mb-5">
            <h4 className="text-base font-bold tracking-widest text-ink">站点 SEO</h4>
            <p className="mt-2 text-sm leading-relaxed text-ink-light opacity-80">
              用于页面标题、默认描述、关键词以及 RSS/Sitemap 链接生成。
            </p>
          </div>
          <div className="grid grid-cols-1 gap-5 md:grid-cols-2">
            <label className="block text-sm text-ink-light">
              <span className="mb-2 block tracking-widest">站点标题</span>
              <input
                value={draftSiteTitle}
                onChange={(event) => setDraftSiteTitle(event.target.value)}
                disabled={isLoading}
                className="w-full border border-mountain-grey bg-transparent px-3 py-2 text-ink outline-none focus:border-ochre disabled:cursor-not-allowed disabled:opacity-50"
              />
            </label>
            <label className="block text-sm text-ink-light">
              <span className="mb-2 block tracking-widest">站点地址</span>
              <input
                value={draftSiteBaseUrl}
                onChange={(event) => setDraftSiteBaseUrl(event.target.value)}
                disabled={isLoading}
                placeholder="https://example.com"
                className="w-full border border-mountain-grey bg-transparent px-3 py-2 text-ink outline-none focus:border-ochre disabled:cursor-not-allowed disabled:opacity-50"
              />
            </label>
            <label className="block text-sm text-ink-light md:col-span-2">
              <span className="mb-2 block tracking-widest">站点描述</span>
              <textarea
                value={draftSiteDescription}
                onChange={(event) => setDraftSiteDescription(event.target.value)}
                disabled={isLoading}
                rows={3}
                className="w-full resize-none border border-mountain-grey bg-transparent px-3 py-2 text-ink outline-none focus:border-ochre disabled:cursor-not-allowed disabled:opacity-50"
              />
            </label>
            <label className="block text-sm text-ink-light md:col-span-2">
              <span className="mb-2 block tracking-widest">关键词</span>
              <input
                value={draftSiteKeywords}
                onChange={(event) => setDraftSiteKeywords(event.target.value)}
                disabled={isLoading}
                placeholder="blog,notes,writing"
                className="w-full border border-mountain-grey bg-transparent px-3 py-2 text-ink outline-none focus:border-ochre disabled:cursor-not-allowed disabled:opacity-50"
              />
            </label>
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
