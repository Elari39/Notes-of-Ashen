export type FetchUserMode = 'silent' | 'strict';

export interface FetchUserFailureDecision {
  clearSession: boolean;
  notifySessionExpired: boolean;
  rethrow: boolean;
}

export interface FetchUserEffects<TUser> {
  setUser: (user: TUser | null) => void;
  setAccessToken: (accessToken: string | null) => void;
  setInitialized: () => void;
  notifySessionExpired: () => void;
}

export interface ExecuteFetchUserOptions<TUser> {
  mode: FetchUserMode;
  accessToken: string | null;
  request: () => Promise<TUser>;
  effects: FetchUserEffects<TUser>;
}

export interface AuthNavigationState {
  accessToken: string | null;
  user: unknown | null;
  isInitialized: boolean;
  isFetching: boolean;
}

// 优先读取 AppError 顶层 status，再兼容 axios response.status；
// 两者都不存在（网络错误/超时）时返回 0。
const httpStatusFromError = (error: unknown): number => {
  if (typeof error !== 'object' || error === null) return 0;
  const status = (error as { status?: unknown }).status;
  if (typeof status === 'number' && Number.isFinite(status) && status > 0) return status;
  const response = (error as { response?: { status?: number } }).response;
  return response?.status ?? 0;
};

export const resolveFetchUserFailure = (
  mode: FetchUserMode,
  status: number,
): FetchUserFailureDecision => {
  const sessionExpired = status === 401 || status === 403;

  return {
    clearSession: sessionExpired,
    notifySessionExpired: sessionExpired,
    rethrow: mode === 'strict',
  };
};

export const executeFetchUser = async <TUser>({
  mode,
  accessToken,
  request,
  effects,
}: ExecuteFetchUserOptions<TUser>): Promise<TUser | null> => {
  if (!accessToken) {
    effects.setUser(null);
    effects.setInitialized();
    return null;
  }

  try {
    const user = await request();
    effects.setUser(user);
    effects.setInitialized();
    return user;
  } catch (error) {
    const decision = resolveFetchUserFailure(mode, httpStatusFromError(error));
    if (decision.clearSession) {
      effects.setUser(null);
      effects.setAccessToken(null);
    }
    effects.setInitialized();
    if (decision.notifySessionExpired) {
      effects.notifySessionExpired();
    }
    if (decision.rethrow) {
      throw error;
    }
    return null;
  }
};

export const shouldNavigateAfterAuth = ({
  accessToken,
  user,
  isInitialized,
  isFetching,
}: AuthNavigationState): boolean => {
  return isInitialized && !isFetching && Boolean(accessToken) && Boolean(user);
};
