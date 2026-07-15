import assert from 'node:assert/strict';
import test from 'node:test';

import { getAIExactErrorMessage } from '../src/utils/aiErrorMessages.ts';

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
