import assert from 'node:assert/strict';
import test from 'node:test';

import { getAIExactErrorMessage } from '../src/utils/aiErrorMessages.ts';
import { AppError, getErrorMessage, isCurrentPasswordRejection, parseJSONBlobError, toAppError } from '../src/utils/error.ts';

test('AI 设置连接错误会映射为稳定中英文提示', () => {
  assert.equal(
    getAIExactErrorMessage('ai provider request failed', 'zh'),
    'AI 服务请求失败，请检查接口格式、地址、密钥和模型',
  );
  assert.equal(
    getAIExactErrorMessage('ai provider request timed out', 'en'),
    'The AI provider request timed out. Check the timeout settings and try again.',
  );
  assert.equal(
    getAIExactErrorMessage('api key is required for unsaved ai endpoint', 'zh'),
    '当前接口格式或地址尚未保存，请填写对应的 API Key',
  );
});

test('AI 探测响应与密钥迁移错误具有明确映射', () => {
  assert.equal(
    getAIExactErrorMessage('ai probe response is not json', 'zh'),
    'AI 服务探测响应不是有效的 JSON',
  );
  assert.equal(
    getAIExactErrorMessage('ai api key needs update', 'en'),
    'The saved API key must be re-entered',
  );
  assert.equal(
    getAIExactErrorMessage('baseUrl is required when AI is enabled', 'zh'),
    '启用 AI 时必须填写 Base URL',
  );
});

test('AppError 重复转换时保留原始 AI 错误标识和状态', () => {
  const first = toAppError(new AppError('ai assistant is disabled', 40301, 403));
  const second = toAppError(first, 'AI 辅助失败');

  assert.equal(first.message, 'AI 辅助未启用，请先在后台 AI 配置中启用。');
  assert.equal(second.message, 'AI 辅助未启用，请先在后台 AI 配置中启用。');
  assert.equal(second.sourceMessage, 'ai assistant is disabled');
  assert.equal(second.code, 40301);
  assert.equal(second.status, 403);
});

test('未知错误重复转换时使用页面提供的场景化兜底文案', () => {
  const first = toAppError(new AppError('unexpected upstream failure'));

  assert.equal(getErrorMessage(first, 'AI 辅助失败'), 'AI 辅助失败');
  assert.equal(first.sourceMessage, 'unexpected upstream failure');
});

test('无效和过期邮箱验证码使用统一提示', () => {
  assert.equal(
    getErrorMessage(new AppError('email code is invalid or expired')),
    '邮箱验证码无效或已过期，请重新获取',
  );
});

test('备份依赖的数据库迁移缺失会给出可操作提示', () => {
  assert.equal(
    getErrorMessage(new AppError('database schema migration is required')),
    '数据库结构未升级，请执行媒体与内容分析迁移后重试。',
  );
});

test('媒体仍被引用时给出可操作提示', () => {
  assert.equal(
    getErrorMessage(new AppError('media asset is still referenced')),
    '图片仍被文章、历史版本、作品或头像引用，移除引用后再删除',
  );
});

test('媒体上传格式错误与 413 响应会给出可操作提示', () => {
  assert.equal(
    getErrorMessage(new AppError('media type is not supported')),
    '不支持该图片格式。请选择 JPG/JPEG、PNG、GIF、WebP 或 AVIF 文件',
  );
  const error = Object.assign(new Error('request failed'), {
    config: { method: 'post' },
    response: { status: 413, data: '' },
  });
  assert.equal(getErrorMessage(error), '上传文件过大，请压缩后重试');
});

test('下载接口的 JSON Blob 错误会恢复为统一错误结构', async () => {
  const blob = new Blob([JSON.stringify({ code: 40123, message: 'current password is incorrect' })], {
    type: 'application/json; charset=utf-8',
  });
  const parsed = await parseJSONBlobError(blob, 'application/json');

  assert.deepEqual(parsed, { code: 40123, message: 'current password is incorrect' });
  assert.equal(isCurrentPasswordRejection(parsed), true);
  const error = Object.assign(new Error('request failed'), {
    config: { method: 'post' },
    response: { status: 401, data: parsed },
  });
  assert.equal(getErrorMessage(error), '当前管理员密码不正确');
});

test('非 JSON Blob 错误保持原始数据供既有回退处理', async () => {
  const blob = new Blob(['not-json'], { type: 'application/octet-stream' });

  assert.equal(await parseJSONBlobError(blob, 'application/octet-stream'), blob);
});
