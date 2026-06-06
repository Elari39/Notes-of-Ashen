import { create } from 'zustand';

export type Language = 'zh' | 'en';
export type ThemePreference = 'system' | 'light' | 'dark';
export type EffectiveTheme = 'light' | 'dark';

const LANGUAGE_KEY = 'notesOfAshen.language';
const THEME_KEY = 'notesOfAshen.theme';

interface PreferenceState {
  language: Language;
  themePreference: ThemePreference;
  effectiveTheme: EffectiveTheme;
  setLanguage: (language: Language) => void;
  toggleLanguage: () => void;
  setThemePreference: (themePreference: ThemePreference) => void;
  toggleTheme: () => void;
  syncSystemTheme: () => void;
  initializePreferences: () => () => void;
}

const isBrowser = () => typeof window !== 'undefined' && typeof document !== 'undefined';

const getSystemTheme = (): EffectiveTheme => {
  if (!isBrowser()) return 'light';
  return window.matchMedia('(prefers-color-scheme: dark)').matches ? 'dark' : 'light';
};

const resolveEffectiveTheme = (themePreference: ThemePreference): EffectiveTheme => {
  return themePreference === 'system' ? getSystemTheme() : themePreference;
};

const readLanguage = (): Language => {
  if (!isBrowser()) return 'zh';
  return localStorage.getItem(LANGUAGE_KEY) === 'en' ? 'en' : 'zh';
};

const readThemePreference = (): ThemePreference => {
  if (!isBrowser()) return 'system';
  const storedTheme = localStorage.getItem(THEME_KEY);
  if (storedTheme === 'light' || storedTheme === 'dark' || storedTheme === 'system') {
    return storedTheme;
  }
  return 'system';
};

const applyLanguage = (language: Language) => {
  if (!isBrowser()) return;
  document.documentElement.lang = language === 'zh' ? 'zh-CN' : 'en';
};

const applyTheme = (theme: EffectiveTheme) => {
  if (!isBrowser()) return;
  document.documentElement.dataset.theme = theme;
  document.documentElement.style.colorScheme = theme;
};

const initialLanguage = readLanguage();
const initialThemePreference = readThemePreference();
const initialEffectiveTheme = resolveEffectiveTheme(initialThemePreference);

applyLanguage(initialLanguage);
applyTheme(initialEffectiveTheme);

export const usePreferenceStore = create<PreferenceState>((set, get) => ({
  language: initialLanguage,
  themePreference: initialThemePreference,
  effectiveTheme: initialEffectiveTheme,
  setLanguage: (language) => {
    if (isBrowser()) {
      localStorage.setItem(LANGUAGE_KEY, language);
    }
    applyLanguage(language);
    set({ language });
  },
  toggleLanguage: () => {
    const nextLanguage = get().language === 'zh' ? 'en' : 'zh';
    get().setLanguage(nextLanguage);
  },
  setThemePreference: (themePreference) => {
    if (isBrowser()) {
      localStorage.setItem(THEME_KEY, themePreference);
    }
    const effectiveTheme = resolveEffectiveTheme(themePreference);
    applyTheme(effectiveTheme);
    set({ themePreference, effectiveTheme });
  },
  toggleTheme: () => {
    const nextTheme = get().effectiveTheme === 'dark' ? 'light' : 'dark';
    get().setThemePreference(nextTheme);
  },
  syncSystemTheme: () => {
    if (get().themePreference !== 'system') return;
    const effectiveTheme = getSystemTheme();
    applyTheme(effectiveTheme);
    set({ effectiveTheme });
  },
  initializePreferences: () => {
    applyLanguage(get().language);
    applyTheme(get().effectiveTheme);

    if (!isBrowser()) {
      return () => undefined;
    }

    const media = window.matchMedia('(prefers-color-scheme: dark)');
    const handleChange = () => get().syncSystemTheme();
    media.addEventListener('change', handleChange);

    return () => {
      media.removeEventListener('change', handleChange);
    };
  },
}));
