import assert from 'node:assert/strict';
import { readFileSync } from 'node:fs';
import test from 'node:test';

const read = (relativePath: string): string => readFileSync(new URL(relativePath, import.meta.url), 'utf8');

test('Service Worker 只缓存 SPA HTML，并排除 XML 与健康探针', () => {
  const source = read('../public/sw.js');
  assert.match(source, /isSpaNavigationPath/);
  assert.match(source, /pathname === '\/rss\.xml'/);
  assert.match(source, /pathname === '\/sitemap\.xml'/);
  assert.match(source, /pathname === '\/healthz'/);
  assert.match(source, /contentType\.toLowerCase\(\)\.includes\('text\/html'\)/);
  assert.match(source, /await cache\.put\('\/index\.html'/);
});

test('项目保存仅提交 tagIds，改密成功后清空会话并跳转登录', () => {
  const projects = read('../src/pages/admin/ProjectsContent.tsx');
  const profile = read('../src/pages/Profile.tsx');
  assert.match(projects, /tagIds: normalizeTagIds/);
  assert.doesNotMatch(projects, /tags: normalizeTags/);
  assert.match(profile, /logout\(\);/);
  assert.match(profile, /navigate\('\/login'/);
});
