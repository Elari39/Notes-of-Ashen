import assert from 'node:assert/strict';
import test from 'node:test';

import { resolveDefaultTimeout } from '../src/utils/timeoutPolicy.ts';

test('AI 提供商请求由服务端设置控制超时', () => {
  assert.equal(resolveDefaultTimeout({ method: 'post', url: '/articles/ai/assist' }), 0);
  assert.equal(resolveDefaultTimeout({ method: 'post', url: '/admin/ai/models' }), 0);
  assert.equal(resolveDefaultTimeout({ method: 'post', url: '/admin/ai/test?draft=1' }), 0);
  assert.equal(resolveDefaultTimeout({ method: 'post', url: '/admin/rag/test' }), 0);
});

test('AI 设置读写仍使用普通请求超时', () => {
  assert.equal(resolveDefaultTimeout({ method: 'get', url: '/admin/ai/settings' }), 10_000);
  assert.equal(resolveDefaultTimeout({ method: 'put', url: '/admin/ai/settings' }), 30_000);
});

test('非 AI 长任务和普通请求保持原有分级策略', () => {
  assert.equal(resolveDefaultTimeout({ method: 'post', url: '/articles/import' }), 600_000);
  assert.equal(resolveDefaultTimeout({ method: 'post', url: '/admin/backups/export' }), 600_000);
  assert.equal(resolveDefaultTimeout({ method: 'post', url: '/admin/backups/restore' }), 600_000);
  assert.equal(resolveDefaultTimeout({ method: 'get', url: '/articles' }), 10_000);
  assert.equal(resolveDefaultTimeout({ method: 'post', url: '/articles' }), 30_000);
});
