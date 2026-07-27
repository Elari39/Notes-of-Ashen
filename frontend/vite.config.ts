import { defineConfig, loadEnv } from 'vite'
import react from '@vitejs/plugin-react'
import tailwindcss from '@tailwindcss/vite'

const chunkGroups: Array<[string, string[]]> = [
  ['react', ['react', 'react-dom', 'scheduler']],
  ['motion', ['framer-motion']],
  ['markdown', ['react-markdown', 'remark-gfm']],
  // KaTeX 独立成 chunk，由 DeferredMarkdownRenderer 仅在内容含公式时加载。
  ['katex', ['katex', 'rehype-katex', 'remark-math']],
  ['mermaid', ['mermaid']],
  ['syntax-highlighter', ['react-syntax-highlighter']],
  ['echarts', ['echarts']],
]

const packagePath = (packageName: string) => `/node_modules/${packageName}/`
const syntaxLanguagePath = '/node_modules/react-syntax-highlighter/dist/esm/languages/'

// https://vitejs.dev/config/
export default defineConfig(({ mode }) => {
  const env = loadEnv(mode, process.cwd(), '')
  const apiTarget = env.VITE_API_TARGET || 'http://127.0.0.1:19000'
  return {
    plugins: [react(), tailwindcss()],
    build: {
      sourcemap: false,
      chunkSizeWarningLimit: 550,
      rollupOptions: {
        output: {
          manualChunks(id) {
            if (!id.includes('node_modules')) {
              return undefined
            }
            const normalized = id.replace(/\\/g, '/')
            // 语言语法由 MarkdownCodeBlock 的动态 import 按需加载，不能并入基础高亮 chunk。
            if (normalized.includes(syntaxLanguagePath)) {
              return undefined
            }
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
          target: apiTarget,
          changeOrigin: true,
        },
        '/media': {
          target: apiTarget,
          changeOrigin: true,
        },
      }
    }
  }
})
