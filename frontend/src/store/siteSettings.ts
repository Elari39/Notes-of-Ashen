import { create } from 'zustand';
import { getSiteSettings, updateSiteSettings } from '../api/siteSettings';
import type { HomeArticleLayout } from '../types';
import type { UpdateSiteSettingsReq } from '../types/api';

interface SiteSettingsState {
  registrationEnabled: boolean;
  registrationEmailCodeRequired: boolean;
  homeArticleLayout: HomeArticleLayout;
  siteTitle: string;
  siteDescription: string;
  siteKeywords: string;
  siteBaseUrl: string;
  resumePageEnabled: boolean;
  resumeNavHidden: boolean;
  projectsPageEnabled: boolean;
  projectsNavHidden: boolean;
  isLoading: boolean;
  hasLoaded: boolean;
  error: string;
  fetchSettings: () => Promise<void>;
  updateSettings: (settings: UpdateSiteSettingsReq) => Promise<void>;
  setRegistrationEnabled: (enabled: boolean) => Promise<void>;
  setHomeArticleLayout: (layout: HomeArticleLayout) => Promise<void>;
}

// 未加载完成时拦截写操作，避免用 store 默认值覆盖后端真实值。
const ensureLoaded = (hasLoaded: boolean): void => {
  if (!hasLoaded) {
    throw new Error('site settings are not loaded yet');
  }
};

export const useSiteSettingsStore = create<SiteSettingsState>((set, get) => ({
  registrationEnabled: true,
  registrationEmailCodeRequired: true,
  homeArticleLayout: 'standard',
  siteTitle: 'Notes of Ashen',
  siteDescription: 'A personal blog written slowly by the lamp of ink.',
  siteKeywords: 'blog,notes,writing',
  siteBaseUrl: '',
  resumePageEnabled: false,
  resumeNavHidden: true,
  projectsPageEnabled: false,
  projectsNavHidden: true,
  isLoading: false,
  hasLoaded: false,
  error: '',
  fetchSettings: async () => {
    set({ isLoading: true, error: '' });
    try {
      const res = await getSiteSettings();
      set({
        registrationEnabled: res.data.registrationEnabled,
        registrationEmailCodeRequired: res.data.registrationEmailCodeRequired ?? true,
        homeArticleLayout: res.data.homeArticleLayout || 'standard',
        siteTitle: res.data.siteTitle || 'Notes of Ashen',
        siteDescription: res.data.siteDescription || '',
        siteKeywords: res.data.siteKeywords || '',
        siteBaseUrl: res.data.siteBaseUrl || '',
        resumePageEnabled: Boolean(res.data.resumePageEnabled),
        resumeNavHidden: res.data.resumeNavHidden ?? true,
        projectsPageEnabled: Boolean(res.data.projectsPageEnabled),
        projectsNavHidden: res.data.projectsNavHidden ?? true,
        hasLoaded: true,
      });
    } catch (error) {
      set({ error: error instanceof Error ? error.message : 'Failed to load site settings', hasLoaded: true });
    } finally {
      set({ isLoading: false });
    }
  },
  updateSettings: async (settings) => {
    ensureLoaded(get().hasLoaded);
    set({ isLoading: true, error: '' });
    try {
      const res = await updateSiteSettings(settings);
      set({
        registrationEnabled: res.data.registrationEnabled,
        registrationEmailCodeRequired: res.data.registrationEmailCodeRequired ?? true,
        homeArticleLayout: res.data.homeArticleLayout || 'standard',
        siteTitle: res.data.siteTitle || 'Notes of Ashen',
        siteDescription: res.data.siteDescription || '',
        siteKeywords: res.data.siteKeywords || '',
        siteBaseUrl: res.data.siteBaseUrl || '',
        resumePageEnabled: Boolean(res.data.resumePageEnabled),
        resumeNavHidden: res.data.resumeNavHidden ?? true,
        projectsPageEnabled: Boolean(res.data.projectsPageEnabled),
        projectsNavHidden: res.data.projectsNavHidden ?? true,
        hasLoaded: true,
      });
    } catch (error) {
      set({ error: error instanceof Error ? error.message : 'Failed to update site settings', hasLoaded: true });
      throw error;
    } finally {
      set({ isLoading: false });
    }
  },
  setRegistrationEnabled: async (enabled) => {
    // 仅发差异字段，后端 UpdateSiteSettingsReq 全部 optional，缺失字段保留当前值。
    await get().updateSettings({ registrationEnabled: enabled });
  },
  setHomeArticleLayout: async (layout) => {
    await get().updateSettings({ homeArticleLayout: layout });
  },
}));
