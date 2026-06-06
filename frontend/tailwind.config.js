/** @type {import('tailwindcss').Config} */
export default {
  content: [
    "./index.html",
    "./src/**/*.{js,ts,jsx,tsx}",
  ],
  theme: {
    extend: {
      colors: {
        paper: '#F9F9F4',    // 宣纸白
        ink: '#1A1A1A',      // 墨黑
        'ink-light': '#425066', // 黛蓝/浅墨
        ochre: '#8A3C3A',    // 赭石 (印章红/点缀)
        'mountain-grey': '#EBEBEB', // 远山灰
      },
      fontFamily: {
        serif: ['"Noto Serif SC"', '"Songti SC"', 'SimSun', '"Times New Roman"', 'serif'],
      },
      spacing: {
        '128': '32rem',
      }
    },
  },
  plugins: [
    require('@tailwindcss/typography'),
  ],
}
