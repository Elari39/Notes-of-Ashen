import axios, {
  type AxiosAdapter,
  type AxiosRequestConfig,
  type AxiosResponse,
  type InternalAxiosRequestConfig,
  getAdapter,
} from 'axios';
import { useAuthStore } from '../store/auth';
import { useUIStore } from '../store/ui';
import { AppError, toAppError } from './error';
import { fixVisibleMojibakeDeep } from './mojibake';
import { notifyFromError } from './notify';

const http = axios.create({
  baseURL: '/api/v1',
  // 不在实例上写死 timeout，由请求拦截器按 method/路径注入分级超时；
  // 业务方仍可在单次调用里显式传 timeout 覆盖。
});

// ---- 分级超时策略 ----
// 后端 RestConf.Timeout = 610s，远高于客户端；写操作链路（MySQL/Redis/MQ 远程）抖动时
// 容易出现"前端超时但服务端实际已写入"。按 method/路径分级，给写请求更宽裕的窗口，
// 同时把 AI/导入导出等长任务接口的窗口拉到 600s。
const TIMEOUT_DEFAULT_GET = 10_000;
const TIMEOUT_DEFAULT_WRITE = 30_000;
const TIMEOUT_LONG_RUNNING = 600_000;

const LONG_RUNNING_PATTERNS: RegExp[] = [
  /\/ai\//,
  /\/articles\/import\b/,
  /\/articles\/[^/]+\/export\b/,
  /\/admin\/search\/reindex\b/,
];

const resolveDefaultTimeout = (config: AxiosRequestConfig): number => {
  const url = config.url ?? '';
  if (LONG_RUNNING_PATTERNS.some((pattern) => pattern.test(url))) {
    return TIMEOUT_LONG_RUNNING;
  }
  const method = (config.method || 'get').toLowerCase();
  return method === 'get' ? TIMEOUT_DEFAULT_GET : TIMEOUT_DEFAULT_WRITE;
};

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

const handleSessionExpired = (refreshError: unknown) => {
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
    return Promise.reject(toAppError(new AppError(data.message || '操作失败，请稍后重试', data.code)));
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
      try {
        const accessToken = await getRefreshTokenTask();
        originalRequest.headers = originalRequest.headers ?? {};
        originalRequest.headers.Authorization = `Bearer ${accessToken}`;
        return http(originalRequest);
      } catch (refreshError) {
        handleSessionExpired(refreshError);
        return Promise.reject(toAppError(refreshError, '登录已过期，请重新登录'));
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
// 同一 method+url+payload 的写请求若已在飞行，后续调用复用同一 Promise，不会真正发出第二个请求。
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
  try {
    return typeof localStorage !== 'undefined' && localStorage.getItem('NOA_DEDUPE') === 'off';
  } catch {
    return false;
  }
};

const buildDedupeKey = (config: AxiosRequestConfig): string | null => {
  if (isDedupeDisabled()) return null;
  const method = (config.method || 'get').toLowerCase();
  if (method === 'get') return null;
  if (config.responseType === 'blob') return null;
  if (config.signal) return null; // 调用方明确管控生命周期，跳过去重
  const url = config.url ?? '';
  if (url.includes('/ai/')) return null;
  if (typeof FormData !== 'undefined' && config.data instanceof FormData) return null;

  let body = '';
  try {
    if (typeof config.data === 'string') {
      body = config.data;
    } else if (typeof config.data === 'undefined' || config.data === null) {
      body = '';
    } else {
      body = stableStringify(config.data);
    }
  } catch {
    return null; // 无法序列化时退化为不去重
  }
  return `${method.toUpperCase()} ${url} ${body}`;
};

const baseAdapter: AxiosAdapter = getAdapter(http.defaults.adapter ?? ['xhr', 'http', 'fetch']);

http.defaults.adapter = (async (config: InternalAxiosRequestConfig) => {
  const key = buildDedupeKey(config);
  if (!key) {
    return baseAdapter(config);
  }
  const existing = inflightWrites.get(key);
  if (existing) {
    // 注意：复用 Promise 时请求拦截器并未再次执行，incLoading 也不会重复加，
    // 因此 globalLoading 计数与 toast 行为保持一致。
    return existing;
  }
  const promise = baseAdapter(config).finally(() => {
    inflightWrites.delete(key);
  });
  inflightWrites.set(key, promise);
  return promise;
}) as AxiosAdapter;

export default http;
