import assert from 'node:assert/strict';
import { readFileSync } from 'node:fs';
import test from 'node:test';

import { resolveLogoutFailure } from '../src/store/logoutPolicy.ts';
import { editorDraftKey } from '../src/utils/editorDraft.ts';

test('注销成功或 401 后允许清理会话，网络与 5xx 保留会话', () => {
  assert.deepEqual(resolveLogoutFailure(401), { clearSession: true, retryable: false });
  assert.deepEqual(resolveLogoutFailure(0), { clearSession: false, retryable: true });
  assert.deepEqual(resolveLogoutFailure(500), { clearSession: false, retryable: true });
  assert.deepEqual(resolveLogoutFailure(403), { clearSession: false, retryable: false });
});

test('注销端点不触发 Axios 自动 refresh，避免刷新失败提前清理会话', () => {
  const source = readFileSync(new URL('../src/utils/http.ts', import.meta.url), 'utf8');
  assert.match(source, /url\?\.includes\('\/auth\/logout'\)/);
});

test('编辑草稿键按用户和文章隔离，新文章也不会跨账户复用', () => {
  assert.equal(editorDraftKey('new', 1), 'article-editor:draft:user:1:new');
  assert.equal(editorDraftKey('new', 2), 'article-editor:draft:user:2:new');
  assert.equal(editorDraftKey(42, 1), 'article-editor:draft:user:1:42');
  assert.notEqual(editorDraftKey(42, 1), editorDraftKey(42, 2));
  assert.equal(editorDraftKey('new'), null);
});
