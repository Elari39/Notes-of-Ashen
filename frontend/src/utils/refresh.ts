import axios from 'axios';
import { AppError } from './error';

type RefreshTokenResp = {
  code: number;
  message?: string;
  data?: {
    accessToken?: string;
    refreshToken?: string;
  };
};

// 与 utils/http.ts 的分级超时常量保持一致：刷新请求单独给一个窗口。
const TIMEOUT_REFRESH_TOKEN = 15_000;

// refreshToken 由后端 HttpOnly Cookie 携带，前端无法读取；accessToken 仅存内存（store）。
// 刷新失败（无 Cookie / Cookie 过期 / 网络错误）一律视为会话失效。
//
// 单独放在此模块，避免 store/auth.ts 与 utils/http.ts 互相 import 形成循环依赖：
// http.ts 在拦截器里调用本函数，store 也在 initializeAuth 里调用。
export const refreshAccessToken = async (): Promise<string> => {
  const res = await axios.post<RefreshTokenResp>('/api/v1/auth/refresh', {}, {
    timeout: TIMEOUT_REFRESH_TOKEN,
    withCredentials: true,
  });
  if (res.data.code !== 0 || !res.data.data?.accessToken) {
    throw new AppError(res.data.message || '登录已过期，请重新登录', res.data.code, res.status);
  }
  return res.data.data.accessToken;
};
