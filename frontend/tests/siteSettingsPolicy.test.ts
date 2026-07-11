import assert from 'node:assert/strict';
import test from 'node:test';

import {
  areSiteSettingsControlsDisabled,
  canWriteSiteSettings,
  executeFetchSettings,
  executeUpdateSettings,
  resolvePublicFeatureRoute,
} from '../src/store/siteSettingsPolicy.ts';

interface TestState {
  isLoading: boolean;
  hasLoaded: boolean;
  loadError: string;
  siteTitle: string;
}

const createStateHarness = () => {
  const state: TestState = {
    isLoading: false,
    hasLoaded: false,
    loadError: '',
    siteTitle: 'default',
  };

  return {
    state,
    setState: (patch: Partial<TestState>) => Object.assign(state, patch),
    toState: (data: { siteTitle: string }): Partial<TestState> => ({ siteTitle: data.siteTitle }),
  };
};

test('首次加载失败后可重试成功，并清理加载错误、更新真实值', async () => {
  const harness = createStateHarness();
  let attempts = 0;
  const request = async () => {
    attempts += 1;
    if (attempts === 1) {
      throw new Error('network error');
    }
    return { siteTitle: 'loaded title' };
  };

  await executeFetchSettings({ ...harness, request });
  assert.deepEqual(harness.state, {
    isLoading: false,
    hasLoaded: false,
    loadError: 'network error',
    siteTitle: 'default',
  });

  await executeFetchSettings({ ...harness, request });
  assert.deepEqual(harness.state, {
    isLoading: false,
    hasLoaded: true,
    loadError: '',
    siteTitle: 'loaded title',
  });
});

test('保存失败保持已加载状态与加载错误，并可直接重试成功', async () => {
  const harness = createStateHarness();
  harness.setState({ hasLoaded: true, loadError: '', siteTitle: 'before' });
  let attempts = 0;
  const request = async () => {
    attempts += 1;
    if (attempts === 1) {
      throw new Error('save error');
    }
    return { siteTitle: 'after' };
  };

  await assert.rejects(
    executeUpdateSettings({ ...harness, hasLoaded: harness.state.hasLoaded, request }),
    /save error/,
  );
  assert.deepEqual(harness.state, {
    isLoading: false,
    hasLoaded: true,
    loadError: '',
    siteTitle: 'before',
  });

  await executeUpdateSettings({ ...harness, hasLoaded: harness.state.hasLoaded, request });
  assert.deepEqual(harness.state, {
    isLoading: false,
    hasLoaded: true,
    loadError: '',
    siteTitle: 'after',
  });
});

test('站点设置未加载时保存不会调用 API', async () => {
  const harness = createStateHarness();
  let calls = 0;

  await assert.rejects(
    executeUpdateSettings({
      ...harness,
      hasLoaded: false,
      request: async () => {
        calls += 1;
        return { siteTitle: 'unexpected' };
      },
    }),
    /not loaded/,
  );
  assert.equal(calls, 0);
});

test('站点设置已加载时允许写入，不受上一次保存错误影响', () => {
  assert.equal(canWriteSiteSettings(true), true);
});

test('后台设置仅在已加载且非请求中时启用表单', () => {
  assert.equal(areSiteSettingsControlsDisabled(false, false), true);
  assert.equal(areSiteSettingsControlsDisabled(true, true), true);
  assert.equal(areSiteSettingsControlsDisabled(true, false), false);
});

test('站点设置加载失败时显示错误，而不是继续加载或返回 404', () => {
  assert.equal(resolvePublicFeatureRoute({
    hasLoaded: false,
    isLoading: false,
    loadError: 'network error',
    enabled: false,
  }), 'error');
});

test('功能关闭仅在站点设置成功加载后返回 404', () => {
  assert.equal(resolvePublicFeatureRoute({
    hasLoaded: true,
    isLoading: false,
    loadError: '',
    enabled: false,
  }), 'notFound');
});

test('初次加载和重试期间都显示加载状态', () => {
  assert.equal(resolvePublicFeatureRoute({
    hasLoaded: false,
    isLoading: false,
    loadError: '',
    enabled: false,
  }), 'loading');
  assert.equal(resolvePublicFeatureRoute({
    hasLoaded: false,
    isLoading: true,
    loadError: '',
    enabled: false,
  }), 'loading');
});

test('站点设置已加载且功能开启时显示页面内容', () => {
  assert.equal(resolvePublicFeatureRoute({
    hasLoaded: true,
    isLoading: false,
    loadError: '',
    enabled: true,
  }), 'content');
});
