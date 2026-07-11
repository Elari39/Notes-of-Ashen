import { create } from 'zustand';
import { User } from '../types';
import { getCurrentUser } from '../api/user';
import { refreshAccessToken } from '../utils/refresh';

// 优先读取 AppError 顶层 status，再兼容 axios response.status；
// 两者都不存在（网络错误/超时）时返回 0。
const httpStatusFromError = (error: unknown): number => {
  if (typeof error !== 'object' || error === null) return 0;
  const status = (error as { status?: unknown }).status;
  if (typeof status === 'number' && Number.isFinite(status) && status > 0) return status;
  const response = (error as { response?: { status?: number } }).response;
  return response?.status ?? 0;
};

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
  fetchUser: () => Promise<void>;
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
  fetchUser: async () => {
    if (!get().accessToken) {
      set({ user: null, isInitialized: true, isFetching: false });
      return;
    }
    set({ isFetching: true });
    try {
      const res = await getCurrentUser();
      set({ user: res.data, isInitialized: true });
    } catch (error) {
      // 仅 401/403（会话真正失效）才清凭证；网络抖动/超时/5xx 保留 token，
      // 避免瞬断把刚刷新到的有效 token 清空强制登出（P4-16）。
      const status = httpStatusFromError(error);
      if (status === 401 || status === 403) {
        set({ user: null, accessToken: null, isInitialized: true });
        get().onSessionExpired?.();
      } else {
        // 网络错误等：保留 token，仅标记初始化完成，UI 可重试。
        set({ isInitialized: true });
      }
    } finally {
      set({ isFetching: false });
    }
  },
  initializeAuth: async () => {
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
  },
}));
