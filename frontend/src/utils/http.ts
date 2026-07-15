import axios, {
  type AxiosAdapter,
  type AxiosRequestConfig,
  type AxiosResponse,
  type InternalAxiosRequestConfig,
  getAdapter,
} from 'axios';
import { useAuthStore } from '../store/auth';
import { useUIStore } from '../store/ui';
import { AppError, ERROR_KEYS, toAppError } from './error';
import { fixVisibleMojibakeDeep } from './mojibake';
import { notifyFromError } from './notify';
import { refreshAccessToken } from './refresh';
import { resolveDefaultTimeout } from './timeoutPolicy';
import { getVisitorId } from './visitor';
import { safeLocalStorage } from './storage';

const http = axios.create({
  baseURL: '/api/v1',
  // refreshToken 已迁移到后端 HttpOnly Cookie，刷新请求需携带凭证。
  withCredentials: true,
  // 不在实例上写死 timeout，由请求拦截器按 method/路径注入分级超时；
  // 业务方仍可在单次调用里显式传 timeout 覆盖。
});

// ---- 分级超时策略 ----
// 后端 RestConf.Timeout = 610s，写操作链路（MySQL/Redis/MQ 远程）抖动时
// 容易出现"前端超时但服务端实际已写入"。按 method/路径分级，给写请求更宽裕的窗口，
// 导入导出等长任务保留 600s；AI 提供商请求返回 0，由后台 AI 设置和服务端安全上限控制。

let refreshTokenTask: Promise<string> | null = null;
let sessionExpiredHandled = false;

const resetSessionExpiredGuard = () => {
  sessionExpiredHandled = false;
};

if (typeof window !== 'undefined') {
  window.addEventListener('noa:auth-changed', resetSessionExpiredGuard);
}

const isAuthRetryEndpoint = (url?: string) => {
  return Boolean(
    url?.includes('/auth/login') ||
    url?.includes('/auth/register') ||
    url?.includes('/auth/refresh'),
  );
};

// 仅对幂等 GET 的网络错误重试，避免写操作断网重复提交
const shouldNetworkRetry = (config?: AxiosRequestConfig) => {
  if (!config) return false;
  const method = (config.method || 'get').toLowerCase();
  if (method !== 'get') return false;
  if (config.responseType === 'blob') return false;
  if (typeof config.url === 'string' && config.url.includes('/ai/')) return false;
  const contentType = typeof config.data === 'undefined' ? '' : 'multipart';
  return contentType !== 'multipart';
};

// 刷新 accessToken：成功后同步更新内存 store。
// 实际的 HTTP 调用放在 utils/refresh.ts 以避免与 store/auth.ts 的循环依赖。
const refreshAccessTokenInStore = async (): Promise<string> => {
  const accessToken = await refreshAccessToken();
  resetSessionExpiredGuard();
  useAuthStore.setState({ accessToken, isInitialized: true });
  return accessToken;
};

const getRefreshTokenTask = () => {
  if (!refreshTokenTask) {
    refreshTokenTask = refreshAccessTokenInStore().finally(() => {
      refreshTokenTask = null;
    });
  }
  return refreshTokenTask;
};

const handleSessionExpired = (refreshError: unknown) => {
  if (sessionExpiredHandled) return;
  sessionExpiredHandled = true;
  useAuthStore.getState().logout();
  notifyFromError(refreshError, 'toast.sessionExpired');
  const handler = useAuthStore.getState().onSessionExpired;
  if (handler) {
    handler();
  }
};

http.interceptors.request.use((config) => {
  useUIStore.getState().incLoading();
  const token = useAuthStore.getState().accessToken;
  if (token && config.headers) {
    config.headers.Authorization = `Bearer ${token}`;
  }
  // 全局注入持久化 visitor id，用于点赞等场景的去重 hash，防换 UA 刷赞。
  const visitorId = getVisitorId();
  if (visitorId && config.headers) {
    config.headers['X-Visitor-Id'] = visitorId;
  }
  // 调用方未显式指定 timeout 时按 method/路径注入分级默认值；显式传入 0 也视为未指定。
  if (config.timeout == null || config.timeout === 0) {
    config.timeout = resolveDefaultTimeout(config);
  }
  return config;
});

http.interceptors.response.use(
  (response) => {
    useUIStore.getState().decLoading();
    if (response.config.responseType === 'blob') {
      return response;
    }
    const data = fixVisibleMojibakeDeep(response.data);
    if (data.code === 0) {
      return data;
    }
    return Promise.reject(toAppError(new AppError(data.message || ERROR_KEYS.operationFailed, data.code)));
  },
  async (error) => {
    useUIStore.getState().decLoading();
    const originalRequest = error.config;

    // 请求被主动取消（AbortController）：不重试、不提示
    if (axios.isCancel(error)) {
      return Promise.reject(error);
    }

    if (error.response?.data) {
      error.response.data = fixVisibleMojibakeDeep(error.response.data);
    }

    if (isAuthRetryEndpoint(originalRequest?.url)) {
      return Promise.reject(toAppError(error));
    }

    if (error.response?.status === 401 && originalRequest && !originalRequest._retry) {
      originalRequest._retry = true;
      // 旋转竞态修复：后端 refresh 即旋转 token。若本请求所用旧 token 与 store 中最新
      // token 不一致，说明 refresh 已被其他并发 401 触发完成，直接换 header 用最新 token
      // 重发即可，无需再触发一次 refresh（避免用旧 token 刷新失败导致强制登出）。
      const failedToken = originalRequest.headers?.Authorization;
      const currentToken = useAuthStore.getState().accessToken;
      if (failedToken && currentToken && failedToken !== `Bearer ${currentToken}`) {
        originalRequest.headers = originalRequest.headers ?? {};
        originalRequest.headers.Authorization = `Bearer ${currentToken}`;
        return http(originalRequest);
      }
      try {
        const accessToken = await getRefreshTokenTask();
        originalRequest.headers = originalRequest.headers ?? {};
        originalRequest.headers.Authorization = `Bearer ${accessToken}`;
        return http(originalRequest);
      } catch (refreshError) {
        handleSessionExpired(refreshError);
        return Promise.reject(toAppError(refreshError, ERROR_KEYS.sessionExpired));
      }
    }

    // 网络错误/超时：对幂等 GET 自动重试一次
    const isNetworkError = !error.response && (error.code === 'ECONNABORTED' || error.code === 'ERR_NETWORK');
    if (isNetworkError && originalRequest && !originalRequest._networkRetry && shouldNetworkRetry(originalRequest)) {
      originalRequest._networkRetry = true;
      try {
        return await http(originalRequest);
      } catch (retryError) {
        return Promise.reject(toAppError(retryError));
      }
    }

    return Promise.reject(toAppError(error));
  },
);

// ---- 写操作 in-flight 去重 ----
// 现网链路抖动时，用户容易在等待中重复点击导致重复提交。这里在 axios adapter 层做去重：
// 同一 method+完整 URI+payload+Authorization 的写请求若已在飞行，后续调用复用同一 Promise，
// 不会真正发出第二个请求。
// 仅作用于非 GET、非 blob、非 FormData、未带 signal、非 /ai/ 的请求；调用方完全无感。
const inflightWrites = new Map<string, Promise<AxiosResponse>>();

const stableStringify = (val: unknown): string => {
  if (val === null || typeof val !== 'object') return JSON.stringify(val) ?? '';
  if (Array.isArray(val)) return `[${val.map(stableStringify).join(',')}]`;
  const entries = Object.entries(val as Record<string, unknown>)
    .filter(([, v]) => typeof v !== 'undefined')
    .sort(([a], [b]) => (a < b ? -1 : a > b ? 1 : 0));
  return `{${entries.map(([k, v]) => `${JSON.stringify(k)}:${stableStringify(v)}`).join(',')}}`;
};

const isDedupeDisabled = () => {
  return safeLocalStorage.getItem('NOA_DEDUPE') === 'off';
};

const getAuthorizationHeader = (headers: AxiosRequestConfig['headers']): string => {
  if (!headers) return '';

  const get = (headers as { get?: (name: string) => unknown }).get;
  const value = typeof get === 'function'
    ? get.call(headers, 'Authorization')
    : Reflect.get(headers, 'Authorization') ?? Reflect.get(headers, 'authorization');

  if (value === null || typeof value === 'undefined') return '';
  return Array.isArray(value) ? value.join(',') : String(value);
};

const buildDedupeKey = (config: AxiosRequestConfig): string | null => {
  if (isDedupeDisabled()) return null;
  // 401 / 网络重试请求虽与原请求同 method+url+body，但 Authorization header 已换，
  // 不能复用 in-flight 的原 Promise，否则重试机制失效。重试请求跳过去重独立发出。
  const retryConfig = config as InternalAxiosRequestConfig & { _retry?: boolean; _networkRetry?: boolean };
  if (retryConfig._retry || retryConfig._networkRetry) return null;
  const method = (config.method || 'get').toLowerCase();
  if (method === 'get') return null;
  if (config.responseType === 'blob') return null;
  if (config.signal) return null; // 调用方明确管控生命周期，跳过去重
  const url = config.url ?? '';
  if (url.includes('/ai/')) return null;
  if (typeof FormData !== 'undefined' && config.data instanceof FormData) return null;

  let body = '';
  let uri = '';
  try {
    if (typeof config.data === 'string') {
      body = config.data;
    } else if (typeof config.data === 'undefined' || config.data === null) {
      body = '';
    } else {
      body = stableStringify(config.data);
    }
    uri = http.getUri(config);
  } catch {
    return null; // 无法序列化时退化为不去重
  }
  const authorization = getAuthorizationHeader(config.headers);
  return JSON.stringify([method.toUpperCase(), uri, body, authorization]);
};

const baseAdapter: AxiosAdapter = getAdapter(http.defaults.adapter ?? ['xhr', 'http', 'fetch']);

http.defaults.adapter = (async (config: InternalAxiosRequestConfig) => {
  const key = buildDedupeKey(config);
  if (!key) {
    return baseAdapter(config);
  }
  const existing = inflightWrites.get(key);
  if (existing) {
    // 每次 http() 调用仍会执行各自的请求/响应拦截器，因此 loading 计数保持成对增减；
    // adapter 层只复用实际网络请求的 Promise。
    return existing;
  }
  const promise = baseAdapter(config).finally(() => {
    inflightWrites.delete(key);
  });
  inflightWrites.set(key, promise);
  return promise;
}) as AxiosAdapter;

export default http;
