import assert from 'node:assert/strict';
import { readFileSync } from 'node:fs';
import test from 'node:test';

import { isNoDataResponse, isSuccessResponse } from '../src/utils/response.ts';

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

test('统一响应处理允许成功 NoData 响应，但拒绝把错误响应当成成功', () => {
  assert.equal(isSuccessResponse({ code: 0, message: 'success' }), true);
  assert.equal(isNoDataResponse({ code: 0, message: 'success' }), true);
  assert.equal(isSuccessResponse({ code: 0, message: 'success', data: undefined }), false);
  assert.equal(isSuccessResponse({ code: 40000, message: 'bad request' }), false);
  assert.equal(isNoDataResponse({ code: 40000, message: 'bad request' }), false);
});

test('NoData mutation API 不再声明必填 data', () => {
  const auth = read('../src/api/auth.ts');
  const article = read('../src/api/article.ts');
  const user = read('../src/api/user.ts');
  const taxonomy = `${read('../src/api/category.ts')}\n${read('../src/api/tag.ts')}`;
  const media = read('../src/api/media.ts');
  const traffic = read('../src/api/traffic.ts');

  assert.match(auth, /NoDataResp/);
  assert.match(article, /deleteArticle[\s\S]*NoDataResp/);
  assert.match(user, /updatePassword[\s\S]*NoDataResp/);
  assert.match(taxonomy, /deleteCategory[\s\S]*NoDataResp/);
  assert.match(taxonomy, /deleteTag[\s\S]*NoDataResp/);
  assert.match(media, /deleteMedia[\s\S]*NoDataResp/);
  assert.match(traffic, /NoDataResp/);
});
