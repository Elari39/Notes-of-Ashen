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
        success: 'var(--success)',
        'success-soft': 'var(--success-soft)',
        warning: 'var(--warning)',
        'warning-soft': 'var(--warning-soft)',
        danger: 'var(--danger)',
        'danger-soft': 'var(--danger-soft)',
        info: 'var(--info)',
        'info-soft': 'var(--info-soft)',
      },
      fontFamily: {
        serif: ['"Noto Serif SC"', '"Songti SC"', 'SimSun', '"Times New Roman"', 'serif'],
      },
      spacing: {
        '128': '32rem',
      },
      borderRadius: {
        xs: 'var(--radius-xs)',
        sm: 'var(--radius-sm)',
        md: 'var(--radius-md)',
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
