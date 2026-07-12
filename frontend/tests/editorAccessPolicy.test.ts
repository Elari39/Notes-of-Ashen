import assert from 'node:assert/strict';
import test from 'node:test';

import {
  canOperateArticleEditor,
  canSaveAISettings,
  executeAISettingsUpdate,
  executeArticleEditorOperation,
  resolveArticleEditorAccess,
} from '../src/pages/admin/editorAccessPolicy.ts';

test('AI 设置首次读取未成功时不会调用更新接口', async () => {
  let calls = 0;

  await assert.rejects(
    executeAISettingsUpdate(false, async () => {
      calls += 1;
      return 'unexpected';
    }),
    /not loaded/,
  );

  assert.equal(calls, 0);
  assert.equal(canSaveAISettings(false), false);
});

test('AI 设置成功读取后，即使一次保存失败仍保持可编辑', async () => {
  await assert.rejects(
    executeAISettingsUpdate(true, async () => {
      throw new Error('save failed');
    }),
    /save failed/,
  );

  assert.equal(canSaveAISettings(true), true);
});

test('既有文章未取得基线时仅进入加载或失败态，且不调用写操作', async () => {
  assert.equal(resolveArticleEditorAccess(true, 'loading'), 'loading');
  assert.equal(resolveArticleEditorAccess(true, 'error'), 'error');
  assert.equal(resolveArticleEditorAccess(true, 'ready'), 'editor');

  let calls = 0;
  await assert.rejects(
    executeArticleEditorOperation(true, 'error', async () => {
      calls += 1;
      return 'unexpected';
    }),
    /baseline is not loaded/,
  );

  assert.equal(calls, 0);
  assert.equal(canOperateArticleEditor(true, 'error'), false);
});

test('新建文章不依赖既有文章基线', () => {
  assert.equal(resolveArticleEditorAccess(false, 'loading'), 'editor');
  assert.equal(canOperateArticleEditor(false, 'loading'), true);
});
