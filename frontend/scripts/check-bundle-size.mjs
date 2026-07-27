/* global console, process */

import { existsSync, readdirSync, readFileSync, statSync } from 'node:fs';
import { join, resolve } from 'node:path';
import { brotliCompressSync, gzipSync } from 'node:zlib';

const distDir = resolve(process.cwd(), 'dist');
const assetsDir = join(distDir, 'assets');

const fail = (message) => {
  throw new Error(`bundle size check failed: ${message}`);
};

if (!existsSync(distDir) || !existsSync(assetsDir)) {
  fail('dist assets are missing; run vite build first');
}

const assets = readdirSync(assetsDir).filter((file) => statSync(join(assetsDir, file)).isFile());
const findChunk = (prefix) => {
  const matches = assets.filter((file) => file.startsWith(`${prefix}-`) && file.endsWith('.js'));
  if (matches.length === 0) {
    fail(`expected a ${prefix} JavaScript chunk, found none`);
  }
  // 语言按需加载后，`markdown` 等语言模块可能与手工 chunk 共享前缀；主 chunk 始终是其中最大的文件。
  return matches.sort((left, right) => statSync(join(assetsDir, right)).size - statSync(join(assetsDir, left)).size)[0];
};
const bytesOf = (file) => readFileSync(join(assetsDir, file));
const toKiB = (bytes) => `${(bytes / 1024).toFixed(2)} KiB`;
const measure = (bytes) => ({
  raw: bytes.length,
  gzip: gzipSync(bytes).length,
  brotli: brotliCompressSync(bytes).length,
});
const assertBudgets = (label, sizes, limits) => {
  for (const [format, limit] of Object.entries(limits)) {
    if (sizes[format] > limit) {
      fail(`${label} ${format} is ${toKiB(sizes[format])}, limit is ${toKiB(limit)}`);
    }
  }
};

const entry = findChunk('index');
const markdown = findChunk('markdown');
const katex = findChunk('katex');
const mermaidRenderer = findChunk('MermaidCodeBlock');
const syntaxHighlighter = findChunk('syntax-highlighter');
const echarts = findChunk('echarts');

const chunkBudgets = [
  {
    label: 'initial JavaScript',
    file: entry,
    limits: { raw: 320 * 1024, gzip: 110 * 1024, brotli: 96 * 1024 },
  },
  {
    // KaTeX 拆出后 markdown chunk 只含 react-markdown/remark-gfm 及关联依赖。
    label: 'markdown chunk',
    file: markdown,
    limits: { raw: 185 * 1024, gzip: 56 * 1024, brotli: 48 * 1024 },
  },
  {
    // 仅当文章内容包含数学公式时由 DeferredMarkdownRenderer 按需加载。
    label: 'katex chunk',
    file: katex,
    limits: { raw: 310 * 1024, gzip: 96 * 1024, brotli: 80 * 1024 },
  },
  {
    // Mermaid 内部的图表实现继续由其自身的动态 import 拆分，渲染器也不应回到 3 MB 单体。
    label: 'mermaid renderer chunk',
    file: mermaidRenderer,
    limits: { raw: 700 * 1024, gzip: 220 * 1024, brotli: 180 * 1024 },
  },
  {
    label: 'syntax-highlighter chunk',
    file: syntaxHighlighter,
    limits: { raw: 64 * 1024, gzip: 24 * 1024, brotli: 20 * 1024 },
  },
  {
    label: 'echarts chunk',
    file: echarts,
    limits: { raw: 550 * 1024, gzip: 185 * 1024, brotli: 155 * 1024 },
  },
];

const reports = chunkBudgets.map(({ label, file, limits }) => {
  const sizes = measure(bytesOf(file));
  assertBudgets(label, sizes, limits);
  return `${label} (${file}): raw ${toKiB(sizes.raw)}, gzip ${toKiB(sizes.gzip)}, brotli ${toKiB(sizes.brotli)}`;
});

const indexHTML = readFileSync(join(distDir, 'index.html'), 'utf8');
const initialAssets = [...indexHTML.matchAll(/\b(?:src|href)="([^"]+)"/g)].map((match) => match[1]);
for (const chunk of [markdown, katex, mermaidRenderer, syntaxHighlighter, echarts]) {
  if (initialAssets.some((asset) => asset.endsWith(`/assets/${chunk}`) || asset.endsWith(`assets/${chunk}`))) {
    fail(`${chunk} must not be an initial index.html asset`);
  }
}

console.log(`bundle size check passed:\n${reports.join('\n')}`);
