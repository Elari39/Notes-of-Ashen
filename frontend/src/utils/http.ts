import axios from 'axios';
import { useAuthStore } from '../store/auth';
import { AppError, toAppError } from './error';

const http = axios.create({
  baseURL: '/api/v1',
  timeout: 10000,
});

// 请求拦截器
http.interceptors.request.use((config) => {
  const token = useAuthStore.getState().accessToken;
  if (token && config.headers) {
    config.headers.Authorization = `Bearer ${token}`;
  }
  return config;
});

// 响应拦截器
http.interceptors.response.use(
  (response) => {
    // 根据通用响应格式解包数据
    if (response.data.code === 0) {
      return response.data;
    }
    return Promise.reject(toAppError(new AppError(response.data.message || '操作失败，请稍后重试', response.data.code)));
  },
  async (error) => {
    const originalRequest = error.config;
    
    // 如果是登录或注册接口报 401，直接返回错误，不进行刷新重试
    if (originalRequest?.url?.includes('/auth/login') || originalRequest?.url?.includes('/auth/register')) {
      return Promise.reject(toAppError(error));
    }

    // 简单处理 401 自动刷新
    if (error.response?.status === 401 && originalRequest && !originalRequest._retry) {
      originalRequest._retry = true;
      try {
        const refreshToken = localStorage.getItem('refreshToken');
        if (!refreshToken) throw new Error('No refresh token');
        
        const res = await axios.post('/api/v1/auth/refresh', { refreshToken });
        if (res.data.code === 0) {
          const { accessToken, refreshToken: newRefresh } = res.data.data;
          localStorage.setItem('accessToken', accessToken);
          localStorage.setItem('refreshToken', newRefresh);
          useAuthStore.setState({ accessToken });
          originalRequest.headers.Authorization = `Bearer ${accessToken}`;
          return http(originalRequest);
        }
      } catch (e) {
        useAuthStore.getState().logout();
        return Promise.reject(new AppError('登录已过期，请重新登录', 40100, 401));
      }
    }
    return Promise.reject(toAppError(error));
  }
);

export default http;
