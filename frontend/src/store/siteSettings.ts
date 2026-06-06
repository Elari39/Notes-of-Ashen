import { create } from 'zustand';
import { getSiteSettings, updateSiteSettings } from '../api/siteSettings';
import type { HomeArticleLayout, SiteSettings } from '../types';

interface SiteSettingsState {
  registrationEnabled: boolean;
  homeArticleLayout: HomeArticleLayout;
  siteTitle: string;
  siteDescription: string;
  siteKeywords: string;
  siteBaseUrl: string;
  isLoading: boolean;
  hasLoaded: boolean;
  error: string;
  fetchSettings: () => Promise<void>;
  updateSettings: (settings: SiteSettings) => Promise<void>;
  setRegistrationEnabled: (enabled: boolean) => Promise<void>;
  setHomeArticleLayout: (layout: HomeArticleLayout) => Promise<void>;
}

export const useSiteSettingsStore = create<SiteSettingsState>((set, get) => ({
  registrationEnabled: true,
  homeArticleLayout: 'standard',
  siteTitle: 'Notes of Ashen',
  siteDescription: 'A personal blog written slowly by the lamp of ink.',
  siteKeywords: 'blog,notes,writing',
  siteBaseUrl: '',
  isLoading: false,
  hasLoaded: false,
  error: '',
  fetchSettings: async () => {
    set({ isLoading: true, error: '' });
    try {
      const res = await getSiteSettings();
      set({
        registrationEnabled: res.data.registrationEnabled,
        homeArticleLayout: res.data.homeArticleLayout || 'standard',
        siteTitle: res.data.siteTitle || 'Notes of Ashen',
        siteDescription: res.data.siteDescription || '',
        siteKeywords: res.data.siteKeywords || '',
        siteBaseUrl: res.data.siteBaseUrl || '',
        hasLoaded: true,
      });
    } catch (error) {
      set({ error: error instanceof Error ? error.message : 'Failed to load site settings', hasLoaded: true });
    } finally {
      set({ isLoading: false });
    }
  },
  updateSettings: async (settings) => {
    set({ isLoading: true, error: '' });
    try {
      const res = await updateSiteSettings(settings);
      set({
        registrationEnabled: res.data.registrationEnabled,
        homeArticleLayout: res.data.homeArticleLayout || 'standard',
        siteTitle: res.data.siteTitle || 'Notes of Ashen',
        siteDescription: res.data.siteDescription || '',
        siteKeywords: res.data.siteKeywords || '',
        siteBaseUrl: res.data.siteBaseUrl || '',
        hasLoaded: true,
      });
    } catch (error) {
      set({ error: error instanceof Error ? error.message : 'Failed to update site settings' });
      throw error;
    } finally {
      set({ isLoading: false });
    }
  },
  setRegistrationEnabled: async (enabled) => {
    const { homeArticleLayout, siteTitle, siteDescription, siteKeywords, siteBaseUrl, updateSettings } = get();
    await updateSettings({ registrationEnabled: enabled, homeArticleLayout, siteTitle, siteDescription, siteKeywords, siteBaseUrl });
  },
  setHomeArticleLayout: async (layout) => {
    const { registrationEnabled, siteTitle, siteDescription, siteKeywords, siteBaseUrl, updateSettings } = get();
    await updateSettings({ registrationEnabled, homeArticleLayout: layout, siteTitle, siteDescription, siteKeywords, siteBaseUrl });
  },
}));
