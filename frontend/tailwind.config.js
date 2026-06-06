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
      },
      fontFamily: {
        serif: ['"Noto Serif SC"', '"Songti SC"', 'SimSun', '"Times New Roman"', 'serif'],
      },
      spacing: {
        '128': '32rem',
      },
    },
  },
  plugins: [
    typography,
  ],
}
