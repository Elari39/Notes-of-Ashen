export const themeStyles = ['editorial', 'soft-brutalism', 'japanese-paper', 'swiss'] as const;

export type ThemeStyle = typeof themeStyles[number];

export const resolveThemeStyle = (value: string | null): ThemeStyle => (
  themeStyles.find((style) => style === value) ?? 'editorial'
);

export const themeStyleAccents: Record<ThemeStyle, { light: string; dark: string }> = {
  editorial: { light: '#cc785c', dark: '#d68a70' },
  'soft-brutalism': { light: '#85412b', dark: '#edaa80' },
  'japanese-paper': { light: '#a94335', dark: '#e29380' },
  swiss: { light: '#b8371e', dark: '#ff987e' },
};
