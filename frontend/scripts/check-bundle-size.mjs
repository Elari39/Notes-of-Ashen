/* global console, process */

import { existsSync, readdirSync, readFileSync, statSync } from 'node:fs';
import { join, resolve } from 'node:path';
import { gzipSync } from 'node:zlib';

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
  if (matches.length !== 1) {
    fail(`expected one ${prefix} JavaScript chunk, found ${matches.length}`);
  }
  return matches[0];
};
const bytesOf = (file) => readFileSync(join(assetsDir, file));
const assertAtMost = (label, bytes, limit) => {
  if (bytes > limit) {
    fail(`${label} is ${(bytes / 1024).toFixed(2)} KiB, limit is ${(limit / 1024).toFixed(2)} KiB`);
  }
};

const entry = findChunk('index');
const markdown = findChunk('markdown');
const syntaxHighlighter = findChunk('syntax-highlighter');
const echarts = findChunk('echarts');
const entryBytes = bytesOf(entry);

assertAtMost('initial JavaScript', entryBytes.length, 320 * 1024);
assertAtMost('initial JavaScript gzip', gzipSync(entryBytes).length, 110 * 1024);
assertAtMost('markdown chunk', bytesOf(markdown).length, 450 * 1024);
assertAtMost('syntax-highlighter chunk', bytesOf(syntaxHighlighter).length, 120 * 1024);
assertAtMost('echarts chunk', bytesOf(echarts).length, 550 * 1024);

const indexHTML = readFileSync(join(distDir, 'index.html'), 'utf8');
const initialAssets = [...indexHTML.matchAll(/\b(?:src|href)="([^"]+)"/g)].map((match) => match[1]);
for (const chunk of [markdown, syntaxHighlighter, echarts]) {
  if (initialAssets.some((asset) => asset.endsWith(`/assets/${chunk}`) || asset.endsWith(`assets/${chunk}`))) {
    fail(`${chunk} must not be an initial index.html asset`);
  }
}

console.log(
  `bundle size check passed: entry ${(entryBytes.length / 1024).toFixed(2)} KiB, ` +
  `markdown ${(bytesOf(markdown).length / 1024).toFixed(2)} KiB, ` +
  `syntax-highlighter ${(bytesOf(syntaxHighlighter).length / 1024).toFixed(2)} KiB, ` +
  `echarts ${(bytesOf(echarts).length / 1024).toFixed(2)} KiB`,
);
