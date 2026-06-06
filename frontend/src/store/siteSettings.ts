import { create } from 'zustand';
import { getSiteSettings, updateSiteSettings } from '../api/siteSettings';
import type { HomeArticleLayout, SiteSettings } from '../types';

interface SiteSettingsState {
  registrationEnabled: boolean;
  homeArticleLayout: HomeArticleLayout;
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
    const { homeArticleLayout, updateSettings } = get();
    await updateSettings({ registrationEnabled: enabled, homeArticleLayout });
  },
  setHomeArticleLayout: async (layout) => {
    const { registrationEnabled, updateSettings } = get();
    await updateSettings({ registrationEnabled, homeArticleLayout: layout });
  },
}));
