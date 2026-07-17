import { create } from 'zustand';
import type { User } from '../types';
import { getCurrentUser } from '../api/user';
import { refreshAccessToken } from '../utils/refresh';
import { executeFetchUser, type FetchUserMode } from './authPolicy';

// React StrictMode 会在开发环境重复执行挂载 effect。静默刷新会旋转 HttpOnly Cookie，
// 因此同一时刻只能保留一个初始化任务，避免两个 refresh 请求竞争同一旧 token。
let initializeAuthTask: Promise<void> | null = null;

interface AuthState {
  user: User | null;
  accessToken: string | null;
  isFetching: boolean;
  isInitialized: boolean;
  /** 会话失效（401 刷新失败）时由 http 层触发，由 App 注入跳转逻辑 */
  onSessionExpired?: () => void;
  setAuth: (user: User | null, token: string | null) => void;
  logout: () => void;
  setSessionExpiredHandler: (handler: (() => void) | undefined) => void;
  fetchUser: (mode?: FetchUserMode) => Promise<User | null>;
  initializeAuth: () => Promise<void>;
}

export const useAuthStore = create<AuthState>((set, get) => ({
  user: null,
  // accessToken 仅存内存：refreshToken 已迁移到后端 HttpOnly Cookie，
  // 刷新页面时通过 /auth/refresh（Cookie）恢复，避免长期凭证落入 localStorage 遭 XSS 窃取。
  accessToken: null,
  isFetching: false,
  isInitialized: false,
  onSessionExpired: undefined,
  setAuth: (user, token) => {
    window.dispatchEvent(new Event('noa:auth-changed'));
    set({ user, accessToken: token, isInitialized: true });
  },
  logout: () => {
    window.dispatchEvent(new Event('noa:auth-changed'));
    set({ user: null, accessToken: null, isInitialized: true, isFetching: false });
  },
  setSessionExpiredHandler: (handler) => {
    set({ onSessionExpired: handler });
  },
  fetchUser: async (mode = 'silent') => {
    const accessToken = get().accessToken;
    if (accessToken) set({ isFetching: true });
    try {
      return await executeFetchUser<User>({
        mode,
        accessToken,
        request: async () => (await getCurrentUser()).data,
        effects: {
          setUser: (user) => set({ user }),
          setAccessToken: (token) => set({ accessToken: token }),
          setInitialized: () => set({ isInitialized: true }),
          notifySessionExpired: () => get().onSessionExpired?.(),
        },
      });
    } finally {
      set({ isFetching: false });
    }
  },
  initializeAuth: () => {
    if (initializeAuthTask) {
      return initializeAuthTask;
    }

    const task = (async () => {
      const { accessToken, user, fetchUser } = get();
      if (accessToken && user) {
        set({ isInitialized: true });
        return;
      }
      // 无内存 token 时尝试通过 HttpOnly Cookie 静默刷新恢复会话；
      // Cookie 不存在或已过期则保持未登录态。
      try {
        const token = await refreshAccessToken();
        set({ accessToken: token, isInitialized: true });
        await fetchUser();
      } catch {
        set({ user: null, accessToken: null, isInitialized: true });
      }
    })();

    const sharedTask = task.finally(() => {
      initializeAuthTask = null;
    });
    initializeAuthTask = sharedTask;
    return sharedTask;
  },
}));
