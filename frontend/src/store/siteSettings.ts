import { create } from 'zustand';
import { getSiteSettings, updateSiteSettings } from '../api/siteSettings';
import type { HomeArticleLayout, SiteSettings } from '../types';

type SiteSettingsUpdateInput = Omit<SiteSettings, 'registrationEmailCodeRequired'>;

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
  updateSettings: (settings: SiteSettingsUpdateInput) => Promise<void>;
  setRegistrationEnabled: (enabled: boolean) => Promise<void>;
  setHomeArticleLayout: (layout: HomeArticleLayout) => Promise<void>;
}

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
      set({ error: error instanceof Error ? error.message : 'Failed to update site settings' });
      throw error;
    } finally {
      set({ isLoading: false });
    }
  },
  setRegistrationEnabled: async (enabled) => {
    const { homeArticleLayout, siteTitle, siteDescription, siteKeywords, siteBaseUrl, resumePageEnabled, resumeNavHidden, projectsPageEnabled, projectsNavHidden, updateSettings } = get();
    await updateSettings({ registrationEnabled: enabled, homeArticleLayout, siteTitle, siteDescription, siteKeywords, siteBaseUrl, resumePageEnabled, resumeNavHidden, projectsPageEnabled, projectsNavHidden });
  },
  setHomeArticleLayout: async (layout) => {
    const { registrationEnabled, siteTitle, siteDescription, siteKeywords, siteBaseUrl, resumePageEnabled, resumeNavHidden, projectsPageEnabled, projectsNavHidden, updateSettings } = get();
    await updateSettings({ registrationEnabled, homeArticleLayout: layout, siteTitle, siteDescription, siteKeywords, siteBaseUrl, resumePageEnabled, resumeNavHidden, projectsPageEnabled, projectsNavHidden });
  },
}));
