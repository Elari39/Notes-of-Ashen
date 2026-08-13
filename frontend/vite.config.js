/* global process */
import { defineConfig, loadEnv } from 'vite';
import react from '@vitejs/plugin-react';
import tailwindcss from '@tailwindcss/vite';
var chunkGroups = [
    ['react', ['react', 'react-dom', 'scheduler']],
    ['motion', ['framer-motion']],
    ['markdown', ['react-markdown', 'remark-gfm']],
    // KaTeX 独立成 chunk，由 DeferredMarkdownRenderer 仅在内容含公式时加载。
    ['katex', ['katex', 'rehype-katex', 'remark-math']],
    ['syntax-highlighter', ['react-syntax-highlighter']],
    ['echarts', ['echarts']],
];
var packagePath = function (packageName) { return "/node_modules/".concat(packageName, "/"); };
var syntaxLanguagePath = '/node_modules/react-syntax-highlighter/dist/esm/languages/';
// https://vitejs.dev/config/
export default defineConfig(function (_a) {
    var mode = _a.mode;
    var env = loadEnv(mode, process.cwd(), '');
    var apiTarget = env.VITE_API_TARGET || 'http://127.0.0.1:19000';
    return {
        plugins: [react(), tailwindcss()],
        build: {
            sourcemap: false,
            chunkSizeWarningLimit: 550,
            rollupOptions: {
                output: {
                    manualChunks: function (id) {
                        if (!id.includes('node_modules')) {
                            return undefined;
                        }
                        var normalized = id.replace(/\\/g, '/');
                        // 语言语法由 MarkdownCodeBlock 的动态 import 按需加载，不能并入基础高亮 chunk。
                        if (normalized.includes(syntaxLanguagePath)) {
                            return undefined;
                        }
                        for (var _i = 0, chunkGroups_1 = chunkGroups; _i < chunkGroups_1.length; _i++) {
                            var _a = chunkGroups_1[_i], chunkName = _a[0], packages = _a[1];
                            if (packages.some(function (packageName) { return normalized.includes(packagePath(packageName)); })) {
                                return chunkName;
                            }
                        }
                        return undefined;
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
    };
});
