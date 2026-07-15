import { create } from 'zustand';
import { getSiteSettings, updateSiteSettings } from '../api/siteSettings';
import type { HomeArticleLayout } from '../types';
import type { SiteSettings } from '../types';
import type { UpdateSiteSettingsReq } from '../types/api';
import { executeFetchSettings, executeUpdateSettings } from './siteSettingsPolicy';

export const DEFAULT_SITE_TITLE = 'Notes of Ashen';
export const DEFAULT_SITE_DESCRIPTION = 'A personal blog written slowly by the lamp of ink.';

interface SiteSettingsState {
  registrationEnabled: boolean;
  registrationEmailCodeRequired: boolean;
  homeArticleLayout: HomeArticleLayout;
  siteTitle: string;
  siteDescription: string;
  siteKeywords: string;
  siteBaseUrl: string;
  projectsPageEnabled: boolean;
  projectsNavHidden: boolean;
  isLoading: boolean;
  hasLoaded: boolean;
  loadError: string;
  fetchSettings: () => Promise<void>;
  updateSettings: (settings: UpdateSiteSettingsReq) => Promise<void>;
  setRegistrationEnabled: (enabled: boolean) => Promise<void>;
  setHomeArticleLayout: (layout: HomeArticleLayout) => Promise<void>;
}

const toSiteSettingsState = (settings: SiteSettings): Partial<SiteSettingsState> => ({
  registrationEnabled: settings.registrationEnabled,
  registrationEmailCodeRequired: settings.registrationEmailCodeRequired ?? true,
  homeArticleLayout: settings.homeArticleLayout || 'standard',
  siteTitle: settings.siteTitle || DEFAULT_SITE_TITLE,
  siteDescription: settings.siteDescription || '',
  siteKeywords: settings.siteKeywords || '',
  siteBaseUrl: settings.siteBaseUrl || '',
  projectsPageEnabled: Boolean(settings.projectsPageEnabled),
  projectsNavHidden: settings.projectsNavHidden ?? true,
});

export const useSiteSettingsStore = create<SiteSettingsState>((set, get) => ({
  registrationEnabled: true,
  registrationEmailCodeRequired: true,
  homeArticleLayout: 'standard',
  siteTitle: DEFAULT_SITE_TITLE,
  siteDescription: DEFAULT_SITE_DESCRIPTION,
  siteKeywords: 'blog,notes,writing',
  siteBaseUrl: '',
  projectsPageEnabled: false,
  projectsNavHidden: true,
  isLoading: false,
  hasLoaded: false,
  loadError: '',
  fetchSettings: () => executeFetchSettings<SiteSettingsState, SiteSettings>({
    request: async () => (await getSiteSettings()).data,
    setState: set,
    toState: toSiteSettingsState,
  }),
  updateSettings: (settings) => executeUpdateSettings<SiteSettingsState, SiteSettings>({
    hasLoaded: get().hasLoaded,
    request: async () => (await updateSiteSettings(settings)).data,
    setState: set,
    toState: toSiteSettingsState,
  }),
  setRegistrationEnabled: async (enabled) => {
    // 仅发差异字段，后端 UpdateSiteSettingsReq 全部 optional，缺失字段保留当前值。
    await get().updateSettings({ registrationEnabled: enabled });
  },
  setHomeArticleLayout: async (layout) => {
    await get().updateSettings({ homeArticleLayout: layout });
  },
}));
