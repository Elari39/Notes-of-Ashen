import { create } from 'zustand';

export type Language = 'zh' | 'en';
export type ThemePreference = 'system' | 'light' | 'dark';
export type EffectiveTheme = 'light' | 'dark';

const LANGUAGE_KEY = 'notesOfAshen.language';
const THEME_KEY = 'notesOfAshen.theme';
const ACCENT_KEY = 'notesOfAshen.accentColor';

interface PreferenceState {
  language: Language;
  themePreference: ThemePreference;
  effectiveTheme: EffectiveTheme;
  accentColor: string;
  setLanguage: (language: Language) => void;
  toggleLanguage: () => void;
  setThemePreference: (themePreference: ThemePreference) => void;
  setAccentColor: (accentColor: string) => void;
  resetAccentColor: () => void;
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

const readAccentColor = () => {
  if (!isBrowser()) return '';
  const value = localStorage.getItem(ACCENT_KEY) || '';
  return /^#[0-9a-f]{6}$/i.test(value) ? value : '';
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

const hexToRgba = (hex: string, alpha: number) => {
  const r = parseInt(hex.slice(1, 3), 16);
  const g = parseInt(hex.slice(3, 5), 16);
  const b = parseInt(hex.slice(5, 7), 16);
  return `rgba(${r}, ${g}, ${b}, ${alpha})`;
};

const applyAccentColor = (accentColor: string) => {
  if (!isBrowser()) return;
  if (accentColor) {
    document.documentElement.style.setProperty('--ochre', accentColor);
    const isDark = document.documentElement.dataset.theme === 'dark';
    document.documentElement.style.setProperty('--inline-code-bg', hexToRgba(accentColor, isDark ? 0.14 : 0.08));
    document.documentElement.style.setProperty('--code-ochre', accentColor);
  } else {
    document.documentElement.style.removeProperty('--ochre');
    document.documentElement.style.removeProperty('--inline-code-bg');
    document.documentElement.style.removeProperty('--code-ochre');
  }
  const themeColor = document.querySelector<HTMLMetaElement>('meta[name="theme-color"]');
  if (themeColor) {
    themeColor.content = accentColor || '#8a3c3a';
  }
};

const initialLanguage = readLanguage();
const initialThemePreference = readThemePreference();
const initialEffectiveTheme = resolveEffectiveTheme(initialThemePreference);
const initialAccentColor = readAccentColor();

applyLanguage(initialLanguage);
applyTheme(initialEffectiveTheme);
applyAccentColor(initialAccentColor);

export const usePreferenceStore = create<PreferenceState>((set, get) => ({
  language: initialLanguage,
  themePreference: initialThemePreference,
  effectiveTheme: initialEffectiveTheme,
  accentColor: initialAccentColor,
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
    // 切主题后重新派生 accent，让 --inline-code-bg 等透明度匹配新主题
    applyAccentColor(get().accentColor);
    set({ themePreference, effectiveTheme });
  },
  setAccentColor: (accentColor) => {
    const normalized = /^#[0-9a-f]{6}$/i.test(accentColor) ? accentColor : '';
    if (isBrowser()) {
      if (normalized) {
        localStorage.setItem(ACCENT_KEY, normalized);
      } else {
        localStorage.removeItem(ACCENT_KEY);
      }
    }
    applyAccentColor(normalized);
    set({ accentColor: normalized });
  },
  resetAccentColor: () => {
    get().setAccentColor('');
  },
  toggleTheme: () => {
    const nextTheme = get().effectiveTheme === 'dark' ? 'light' : 'dark';
    get().setThemePreference(nextTheme);
  },
  syncSystemTheme: () => {
    if (get().themePreference !== 'system') return;
    const effectiveTheme = getSystemTheme();
    applyTheme(effectiveTheme);
    applyAccentColor(get().accentColor);
    set({ effectiveTheme });
  },
  initializePreferences: () => {
    applyLanguage(get().language);
    applyTheme(get().effectiveTheme);
    applyAccentColor(get().accentColor);

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
