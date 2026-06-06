import { create } from 'zustand';
import { getSiteSettings, updateSiteSettings } from '../api/siteSettings';

interface SiteSettingsState {
  registrationEnabled: boolean;
  isLoading: boolean;
  hasLoaded: boolean;
  error: string;
  fetchSettings: () => Promise<void>;
  setRegistrationEnabled: (enabled: boolean) => Promise<void>;
}

export const useSiteSettingsStore = create<SiteSettingsState>((set) => ({
  registrationEnabled: true,
  isLoading: false,
  hasLoaded: false,
  error: '',
  fetchSettings: async () => {
    set({ isLoading: true, error: '' });
    try {
      const res = await getSiteSettings();
      set({ registrationEnabled: res.data.registrationEnabled, hasLoaded: true });
    } catch (error) {
      set({ error: error instanceof Error ? error.message : 'Failed to load site settings', hasLoaded: true });
    } finally {
      set({ isLoading: false });
    }
  },
  setRegistrationEnabled: async (enabled) => {
    set({ isLoading: true, error: '' });
    try {
      const res = await updateSiteSettings({ registrationEnabled: enabled });
      set({ registrationEnabled: res.data.registrationEnabled, hasLoaded: true });
    } catch (error) {
      set({ error: error instanceof Error ? error.message : 'Failed to update site settings' });
      throw error;
    } finally {
      set({ isLoading: false });
    }
  },
}));
