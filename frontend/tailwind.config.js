import typography from '@tailwindcss/typography'

/** @type {import('tailwindcss').Config} */
export default {
  content: [
    './index.html',
    './src/**/*.{js,ts,jsx,tsx}',
  ],
  theme: {
    extend: {
      colors: {
        paper: 'var(--paper)',
        ink: 'var(--ink)',
        'ink-light': 'var(--ink-light)',
        ochre: 'var(--ochre)',
        'mountain-grey': 'var(--mountain-grey)',
        canvas: 'var(--canvas)',
        body: 'var(--body)',
        muted: 'var(--muted)',
        hairline: 'var(--hairline)',
        'surface-soft': 'var(--surface-soft)',
        'surface-card': 'var(--surface-card)',
        'surface-strong': 'var(--surface-strong)',
        'surface-dark': 'var(--surface-dark)',
        'surface-dark-soft': 'var(--surface-dark-soft)',
        'on-dark': 'var(--on-dark)',
        'on-dark-soft': 'var(--on-dark-soft)',
        'on-accent': 'var(--on-accent)',
        // 状态色：moss=苔 / amber=琥珀 / ember=余烬 / dusk=暮色，呼应纸墨灰烬主题
        moss: 'var(--moss)',
        'moss-soft': 'var(--moss-soft)',
        amber: 'var(--amber)',
        'amber-soft': 'var(--amber-soft)',
        ember: 'var(--ember)',
        'ember-soft': 'var(--ember-soft)',
        dusk: 'var(--dusk)',
        'dusk-soft': 'var(--dusk-soft)',
      },
      fontFamily: {
        sans: ['Inter', '"Noto Sans SC"', '"Microsoft YaHei"', 'system-ui', 'sans-serif'],
        display: ['"Cormorant Garamond"', '"Noto Serif SC"', '"Songti SC"', 'SimSun', 'serif'],
        serif: ['"Cormorant Garamond"', '"Noto Serif SC"', '"Songti SC"', 'SimSun', 'serif'],
        mono: ['"JetBrains Mono"', '"SFMono-Regular"', 'Consolas', 'monospace'],
      },
      spacing: {
        '128': '32rem',
      },
      borderRadius: {
        xs: 'var(--radius-xs)',
        sm: 'var(--radius-sm)',
        md: 'var(--radius-md)',
        lg: 'var(--radius-lg)',
        xl: 'var(--radius-xl)',
      },
      boxShadow: {
        xs: 'var(--shadow-xs)',
        sm: 'var(--shadow-sm)',
        md: 'var(--shadow-md)',
        lg: 'var(--shadow-lg)',
      },
      transitionDuration: {
        fast: '120',
        base: '180',
        slow: '280',
        page: '240',
      },
      transitionTimingFunction: {
        paper: 'cubic-bezier(0.22, 1, 0.36, 1)',
      },
      outlineColor: {
        ochre: 'var(--ochre)',
      },
    },
  },
  plugins: [
    typography,
  ],
}
