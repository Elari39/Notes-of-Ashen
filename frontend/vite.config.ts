import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'

const chunkGroups: Array<[string, string[]]> = [
  ['react', ['react', 'react-dom', 'scheduler']],
  ['markdown', ['react-markdown', 'remark-gfm', 'remark-math', 'rehype-katex', 'rehype-slug', 'rehype-autolink-headings', 'katex']],
  ['syntax-highlighter', ['react-syntax-highlighter']],
  ['charts', ['recharts']],
  ['echarts', ['echarts']],
  ['pdf', ['html2pdf.js', 'jspdf', 'html2canvas']],
]

const packagePath = (packageName: string) => `/node_modules/${packageName}/`

// https://vitejs.dev/config/
export default defineConfig({
  plugins: [react()],
  build: {
    chunkSizeWarningLimit: 1000,
    rollupOptions: {
      output: {
        manualChunks(id) {
          if (!id.includes('node_modules')) {
            return undefined
          }
          const normalized = id.replace(/\\/g, '/')
          for (const [chunkName, packages] of chunkGroups) {
            if (packages.some((packageName) => normalized.includes(packagePath(packageName)))) {
              return chunkName
            }
          }
          return undefined
        },
      },
    },
  },
  server: {
    port: 3000,
    proxy: {
      '/api': {
        target: 'http://127.0.0.1:19000',
        changeOrigin: true,
      }
    }
  }
})
