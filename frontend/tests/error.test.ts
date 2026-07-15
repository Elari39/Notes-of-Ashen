import assert from 'node:assert/strict';
import test from 'node:test';

import { getAIExactErrorMessage } from '../src/utils/aiErrorMessages.ts';
import { AppError, getErrorMessage, toAppError } from '../src/utils/error.ts';

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
