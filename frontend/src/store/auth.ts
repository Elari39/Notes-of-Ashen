import { create } from 'zustand';
import { User } from '../types';
import { getCurrentUser } from '../api/user';

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

const storedAccessToken = localStorage.getItem('accessToken');

export const useAuthStore = create<AuthState>((set, get) => ({
  user: null,
  accessToken: storedAccessToken,
  isFetching: false,
  isInitialized: !storedAccessToken,
  onSessionExpired: undefined,
  setAuth: (user, token) => {
    if (token) localStorage.setItem('accessToken', token);
    else localStorage.removeItem('accessToken');
    set({ user, accessToken: token, isInitialized: true });
  },
  logout: () => {
    localStorage.removeItem('accessToken');
    localStorage.removeItem('refreshToken');
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
    } catch {
      localStorage.removeItem('accessToken');
      localStorage.removeItem('refreshToken');
      set({ user: null, accessToken: null, isInitialized: true });
    } finally {
      set({ isFetching: false });
    }
  },
  initializeAuth: async () => {
    const { accessToken, user, fetchUser } = get();
    if (!accessToken || user) {
      set({ isInitialized: true });
      return;
    }
    await fetchUser();
  },
}));
