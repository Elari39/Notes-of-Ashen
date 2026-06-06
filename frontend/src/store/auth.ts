import { create } from 'zustand';
import { User } from '../types';
import { getCurrentUser } from '../api/user';

interface AuthState {
  user: User | null;
  accessToken: string | null;
  isFetching: boolean;
  setAuth: (user: User | null, token: string | null) => void;
  logout: () => void;
  fetchUser: () => Promise<void>;
}

export const useAuthStore = create<AuthState>((set) => ({
  user: null,
  accessToken: localStorage.getItem('accessToken'),
  isFetching: false,
  setAuth: (user, token) => {
    if (token) localStorage.setItem('accessToken', token);
    else localStorage.removeItem('accessToken');
    set({ user, accessToken: token });
  },
  logout: () => {
    localStorage.removeItem('accessToken');
    localStorage.removeItem('refreshToken');
    set({ user: null, accessToken: null });
  },
  fetchUser: async () => {
    set({ isFetching: true });
    try {
      const res = await getCurrentUser();
      set({ user: res.data });
    } catch (e) {
      set({ user: null });
    } finally {
      set({ isFetching: false });
    }
  },
}));
