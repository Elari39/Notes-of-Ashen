export interface LogoutFailureDecision {
  clearSession: boolean;
  retryable: boolean;
}

export const resolveLogoutFailure = (status: number): LogoutFailureDecision => ({
  // 401 明确表示 refresh token 已失效，继续保留本地会话只会造成刷新后恢复旧状态。
  clearSession: status === 401,
  retryable: status === 0 || status >= 500,
});
