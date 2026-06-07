import axios from 'axios';
import { useAuthStore } from '../store/auth';
import { AppError, toAppError } from './error';

const http = axios.create({
  baseURL: '/api/v1',
  timeout: 10000,
});

type RefreshTokenResp = {
  code: number;
  message?: string;
  data?: {
    accessToken?: string;
    refreshToken?: string;
  };
};

let refreshTokenTask: Promise<string> | null = null;

const isAuthRetryEndpoint = (url?: string) => {
  return Boolean(
    url?.includes('/auth/login') ||
    url?.includes('/auth/register') ||
    url?.includes('/auth/refresh'),
  );
};

const refreshAccessToken = async () => {
  const refreshToken = localStorage.getItem('refreshToken');
  if (!refreshToken) {
    throw new AppError('登录已过期，请重新登录', 40100, 401);
  }

  const res = await axios.post<RefreshTokenResp>('/api/v1/auth/refresh', { refreshToken });
  if (res.data.code !== 0 || !res.data.data?.accessToken || !res.data.data.refreshToken) {
    throw new AppError(res.data.message || '登录已过期，请重新登录', res.data.code, res.status);
  }

  const { accessToken, refreshToken: newRefreshToken } = res.data.data;
  localStorage.setItem('accessToken', accessToken);
  localStorage.setItem('refreshToken', newRefreshToken);
  useAuthStore.setState({ accessToken, isInitialized: true });

  return accessToken;
};

const getRefreshTokenTask = () => {
  if (!refreshTokenTask) {
    refreshTokenTask = refreshAccessToken().finally(() => {
      refreshTokenTask = null;
    });
  }
  return refreshTokenTask;
};

http.interceptors.request.use((config) => {
  const token = useAuthStore.getState().accessToken;
  if (token && config.headers) {
    config.headers.Authorization = `Bearer ${token}`;
  }
  return config;
});

http.interceptors.response.use(
  (response) => {
    if (response.config.responseType === 'blob') {
      return response;
    }
    if (response.data.code === 0) {
      return response.data;
    }
    return Promise.reject(toAppError(new AppError(response.data.message || '操作失败，请稍后重试', response.data.code)));
  },
  async (error) => {
    const originalRequest = error.config;

    if (isAuthRetryEndpoint(originalRequest?.url)) {
      return Promise.reject(toAppError(error));
    }

    if (error.response?.status === 401 && originalRequest && !originalRequest._retry) {
      originalRequest._retry = true;
      try {
        const accessToken = await getRefreshTokenTask();
        originalRequest.headers = originalRequest.headers ?? {};
        originalRequest.headers.Authorization = `Bearer ${accessToken}`;
        return http(originalRequest);
      } catch (refreshError) {
        useAuthStore.getState().logout();
        return Promise.reject(toAppError(refreshError, '登录已过期，请重新登录'));
      }
    }
    return Promise.reject(toAppError(error));
  },
);

export default http;
