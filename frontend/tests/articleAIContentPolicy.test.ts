import assert from 'node:assert/strict';
import test from 'node:test';
import { exceedsFullArticleAIContentLimit } from '../src/pages/admin/articleAIContentPolicy.ts';
import { MAX_AI_FULL_ARTICLE_CONTENT_BYTES } from '../src/utils/utf8.ts';

test('全文 AI 操作按 UTF-8 字节限制输入', () => {
  const exactLimit = '你'.repeat(Math.floor(MAX_AI_FULL_ARTICLE_CONTENT_BYTES / 3)) + 'a';
  const overLimit = `${exactLimit}a`;

  assert.equal(exceedsFullArticleAIContentLimit('complete', exactLimit), false);
  assert.equal(exceedsFullArticleAIContentLimit('metadata', overLimit), true);
});

test('改写类 AI 操作不使用全文 64 KiB 限制', () => {
  assert.equal(exceedsFullArticleAIContentLimit('proofread', '你'.repeat(30_000)), false);
});
