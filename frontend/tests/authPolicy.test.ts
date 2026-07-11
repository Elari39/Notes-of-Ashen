import assert from 'node:assert/strict';
import test from 'node:test';

import {
  executeFetchUser,
  resolveFetchUserFailure,
  shouldNavigateAfterAuth,
} from '../src/store/authPolicy.ts';

test('silent 模式遇到 401 时清理会话并通知，但不抛错', () => {
  assert.deepEqual(resolveFetchUserFailure('silent', 401), {
    clearSession: true,
    notifySessionExpired: true,
    rethrow: false,
  });
});

test('strict 模式遇到 403 时清理会话、通知并重新抛错', () => {
  assert.deepEqual(resolveFetchUserFailure('strict', 403), {
    clearSession: true,
    notifySessionExpired: true,
    rethrow: true,
  });
});

test('silent 模式遇到 500 时保留会话且不抛错', () => {
  assert.deepEqual(resolveFetchUserFailure('silent', 500), {
    clearSession: false,
    notifySessionExpired: false,
    rethrow: false,
  });
});

test('strict 模式遇到 500 时保留会话并重新抛错', () => {
  assert.deepEqual(resolveFetchUserFailure('strict', 500), {
    clearSession: false,
    notifySessionExpired: false,
    rethrow: true,
  });
});

test('网络错误沿用非会话失效策略', () => {
  assert.deepEqual(resolveFetchUserFailure('silent', 0), {
    clearSession: false,
    notifySessionExpired: false,
    rethrow: false,
  });
});

interface TestUser {
  id: number;
}

const createStateEffects = (initial: { user: TestUser | null; accessToken: string | null }) => {
  const state = {
    ...initial,
    isInitialized: false,
    notifications: 0,
  };

  return {
    state,
    effects: {
      setUser: (user: TestUser | null) => {
        state.user = user;
      },
      setAccessToken: (accessToken: string | null) => {
        state.accessToken = accessToken;
      },
      setInitialized: () => {
        state.isInitialized = true;
      },
      notifySessionExpired: () => {
        state.notifications += 1;
      },
    },
  };
};

test('strict 401 清理会话、通知并抛出原错误', async () => {
  const { state, effects } = createStateEffects({ user: { id: 1 }, accessToken: 'token' });
  const error = Object.assign(new Error('unauthorized'), { status: 401 });

  await assert.rejects(
    executeFetchUser({
      mode: 'strict',
      accessToken: state.accessToken,
      request: async () => { throw error; },
      effects,
    }),
    error,
  );

  assert.deepEqual(state, {
    user: null,
    accessToken: null,
    isInitialized: true,
    notifications: 1,
  });
});

test('strict 500 保留会话并抛出原错误', async () => {
  const initialUser = { id: 1 };
  const { state, effects } = createStateEffects({ user: initialUser, accessToken: 'token' });
  const error = Object.assign(new Error('server error'), { status: 500 });

  await assert.rejects(
    executeFetchUser({
      mode: 'strict',
      accessToken: state.accessToken,
      request: async () => { throw error; },
      effects,
    }),
    error,
  );

  assert.equal(state.user, initialUser);
  assert.equal(state.accessToken, 'token');
  assert.equal(state.isInitialized, true);
  assert.equal(state.notifications, 0);
});

test('silent 500 保留会话并返回 null', async () => {
  const initialUser = { id: 1 };
  const { state, effects } = createStateEffects({ user: initialUser, accessToken: 'token' });

  const result = await executeFetchUser({
    mode: 'silent',
    accessToken: state.accessToken,
    request: async () => { throw Object.assign(new Error('server error'), { status: 500 }); },
    effects,
  });

  assert.equal(result, null);
  assert.equal(state.user, initialUser);
  assert.equal(state.accessToken, 'token');
  assert.equal(state.isInitialized, true);
  assert.equal(state.notifications, 0);
});

test('请求成功时写入并返回用户', async () => {
  const { state, effects } = createStateEffects({ user: null, accessToken: 'token' });
  const user = { id: 2 };

  const result = await executeFetchUser({
    mode: 'strict',
    accessToken: state.accessToken,
    request: async () => user,
    effects,
  });

  assert.equal(result, user);
  assert.equal(state.user, user);
  assert.equal(state.isInitialized, true);
});

test('无 token 时不请求并返回 null', async () => {
  const { state, effects } = createStateEffects({ user: { id: 1 }, accessToken: null });
  let requestCount = 0;

  const result = await executeFetchUser({
    mode: 'strict',
    accessToken: state.accessToken,
    request: async () => {
      requestCount += 1;
      return { id: 2 };
    },
    effects,
  });

  assert.equal(result, null);
  assert.equal(requestCount, 0);
  assert.equal(state.user, null);
  assert.equal(state.isInitialized, true);
});

test('严格获取失败后的半登录状态不导航', () => {
  assert.equal(shouldNavigateAfterAuth({
    accessToken: 'token',
    user: null,
    isInitialized: true,
    isFetching: false,
  }), false);
});

test('用户获取成功且初始化完成时导航', () => {
  assert.equal(shouldNavigateAfterAuth({
    accessToken: 'token',
    user: { id: 1 },
    isInitialized: true,
    isFetching: false,
  }), true);
});
