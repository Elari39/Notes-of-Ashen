import assert from 'node:assert/strict';
import test from 'node:test';

import {
  buildAIConnectionReq,
  buildAIModelTestReq,
  buildUpdateAISettingsReq,
  canStartAISettingsOperation,
  hasAIConnectionIdentityChanged,
  isAISettingsDirty,
  isAISettingsTimeoutInvalid,
  normalizeAIModels,
  toAISettingsDraft,
  type AISettingsDraft,
} from '../src/pages/admin/aiSettingsPolicy.ts';

const baseline: AISettingsDraft = toAISettingsDraft({
  enabled: true,
  apiFormat: 'openai',
  baseUrl: 'https://api.example.com/v1',
  model: 'model-a',
  apiKeyConfigured: true,
  apiKeyNeedsUpdate: false,
  firstByteTimeoutSeconds: 60,
  nonStreamTimeoutSeconds: 600,
});

test('远端响应会形成不含密钥占位值的可取消基线', () => {
  assert.deepEqual(baseline, {
    enabled: true,
    apiFormat: 'openai',
    baseUrl: 'https://api.example.com/v1',
    model: 'model-a',
    apiKey: '',
    clearApiKey: false,
    firstByteTimeoutSeconds: 60,
    nonStreamTimeoutSeconds: 600,
  });
  assert.equal(isAISettingsDirty({ ...baseline }, baseline), false);
  assert.equal(isAISettingsDirty({ ...baseline, model: 'model-b' }, baseline), true);
  assert.equal(isAISettingsDirty({ ...baseline, apiKey: 'new-secret' }, baseline), true);
});

test('连接探测和保存只发送用户实际输入的密钥', () => {
  assert.deepEqual(buildAIConnectionReq(baseline), {
    apiFormat: 'openai',
    baseUrl: 'https://api.example.com/v1',
    firstByteTimeoutSeconds: 60,
    nonStreamTimeoutSeconds: 600,
  });
  assert.deepEqual(buildAIModelTestReq({ ...baseline, apiKey: '  secret  ' }), {
    apiFormat: 'openai',
    baseUrl: 'https://api.example.com/v1',
    apiKey: 'secret',
    firstByteTimeoutSeconds: 60,
    nonStreamTimeoutSeconds: 600,
    model: 'model-a',
  });
  assert.equal('apiKey' in buildUpdateAISettingsReq(baseline), false);
});

test('连接身份变化会使候选与测试反馈失效，但模型值不属于连接身份', () => {
  assert.equal(hasAIConnectionIdentityChanged(baseline, { ...baseline, apiFormat: 'anthropic' }), true);
  assert.equal(hasAIConnectionIdentityChanged(baseline, { ...baseline, baseUrl: 'https://other.example.com' }), true);
  assert.equal(hasAIConnectionIdentityChanged(baseline, { ...baseline, apiKey: 'secret' }), true);
  assert.equal(hasAIConnectionIdentityChanged(baseline, { ...baseline, model: 'manual-model' }), false);
});

test('超时策略与三类互斥操作可在发请求前判定', () => {
  assert.equal(isAISettingsTimeoutInvalid(baseline), false);
  assert.equal(isAISettingsTimeoutInvalid({ ...baseline, nonStreamTimeoutSeconds: 59 }), true);
  assert.equal(isAISettingsTimeoutInvalid({ ...baseline, firstByteTimeoutSeconds: 1801 }), true);
  assert.equal(canStartAISettingsOperation(null), true);
  assert.equal(canStartAISettingsOperation('models'), false);
});

test('模型候选清理空值并稳定去重，不承担自动选中职责', () => {
  assert.deepEqual(normalizeAIModels([' model-a ', '', 'model-b', 'model-a']), ['model-a', 'model-b']);
});
