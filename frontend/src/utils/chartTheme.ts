export interface ChartThemeColors {
  pv: string;
  uv: string;
  canvas: string;
  ink: string;
  muted: string;
  hairline: string;
}

const fallbackColors: ChartThemeColors = {
  pv: '#cc785c',
  uv: '#5db8a6',
  canvas: '#faf9f5',
  ink: '#141413',
  muted: '#3d3d3a',
  hairline: '#e6dfd8',
};

/**
 * Canvas 图表不能解析 CSS `var()`。从根元素读取已生效的 token，
 * 使浅/深色主题和用户自定义强调色与页面样式保持同步。
 */
export const getChartThemeColors = (): ChartThemeColors => {
  if (typeof window === 'undefined' || typeof document === 'undefined') {
    return fallbackColors;
  }

  const styles = window.getComputedStyle(document.documentElement);
  const read = (token: string, fallback: string) => styles.getPropertyValue(token).trim() || fallback;

  return {
    pv: read('--ochre', fallbackColors.pv),
    uv: read('--accent-teal', fallbackColors.uv),
    canvas: read('--canvas', fallbackColors.canvas),
    ink: read('--ink', fallbackColors.ink),
    muted: read('--body', fallbackColors.muted),
    hairline: read('--hairline', fallbackColors.hairline),
  };
};
