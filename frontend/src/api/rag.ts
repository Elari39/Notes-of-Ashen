import http from '../utils/http';
import { useAuthStore } from '../store/auth';
import { refreshAccessToken } from '../utils/refresh';
import { getVisitorId } from '../utils/visitor';
import type {
  BaseResp,
  RAGChatSession,
  NoDataResp,
} from '../types';
import type {
  RAGChatStreamReq,
  RAGSettingsResp,
  RAGStatusResp,
  RAGTestReq,
  RAGTestResp,
  UpdateRAGSettingsReq,
} from '../types/api';
import { createRAGStreamDecoder, type RAGParsedStreamEvent } from '../utils/ragStream';

export const getRAGSettings = () =>
  http.get<unknown, BaseResp<RAGSettingsResp>>('/admin/rag/settings');

export const updateRAGSettings = (data: UpdateRAGSettingsReq) =>
  http.put<unknown, BaseResp<RAGSettingsResp>>('/admin/rag/settings', data);

export const getRAGStatus = () =>
  http.get<unknown, BaseResp<RAGStatusResp>>('/admin/rag/status');

export const testRAGConnection = (data: RAGTestReq) =>
  http.post<unknown, BaseResp<RAGTestResp>>('/admin/rag/test', data);

export const rebuildRAGIndex = () =>
  http.post<unknown, BaseResp<RAGStatusResp>>('/admin/rag/rebuild');

export const getRAGSessions = (signal?: AbortSignal) =>
  http.get<unknown, BaseResp<{ items: RAGChatSession[] }>>('/rag/sessions', { signal });

export const getRAGSession = (id: string, signal?: AbortSignal) =>
  http.get<unknown, BaseResp<RAGChatSession>>(`/rag/sessions/${encodeURIComponent(id)}`, { signal });

export const deleteRAGSession = (id: string) =>
  http.delete<unknown, NoDataResp>(`/rag/sessions/${encodeURIComponent(id)}`);

export type RAGStreamEvent = RAGParsedStreamEvent;

export type RAGStreamHandlers = {
  onEvent: (event: RAGStreamEvent) => void;
};

/**
 * POST SSE 不走 Axios：浏览器 EventSource 不支持 POST，请求体又不能暴露在 URL 中。
 * 使用 TextDecoder 的 stream 模式，避免中文字符恰好跨网络分块时出现乱码。
 */
export const streamRAGChat = async (
  data: RAGChatStreamReq,
  handlers: RAGStreamHandlers,
  signal?: AbortSignal,
): Promise<void> => {
  const response = await requestRAGStream(data, signal);
  if (!response.ok) {
    throw new Error(await responseErrorMessage(response));
  }
  if (!response.body) {
    throw new Error('RAG stream response body is empty');
  }

  const reader = response.body.getReader();
  let streamError: Error | null = null;
  let streamDone = false;
  let bodyCompleted = false;
  const decoder = createRAGStreamDecoder((event) => {
    handlers.onEvent(event);
    if (event.type === 'error') streamError = new Error(event.message);
    if (event.type === 'done') streamDone = true;
  });

  try {
    for (;;) {
      const readResult = await reader.read();
      if (readResult.done) {
        bodyCompleted = true;
        break;
      }
      decoder.push(readResult.value);
      if (streamError) throw streamError;
    }
    decoder.finish();
    if (streamError) throw streamError;
    // SSE 正常关闭本身不等于回答成功。服务端必须显式发送 done，避免网关、
    // 浏览器或上游提前断流时把局部答案误当成完成答案。
    if (!streamDone) throw new Error('RAG stream ended before completion');
  } finally {
    if (!bodyCompleted) {
      void reader.cancel().catch(() => undefined);
    }
    reader.releaseLock();
  }
};

const requestRAGStream = async (data: RAGChatStreamReq, signal?: AbortSignal): Promise<Response> => {
  const request = async (accessToken: string | null): Promise<Response> => {
    const headers = new Headers({
      Accept: 'text/event-stream',
      'Content-Type': 'application/json',
    });
    if (accessToken) headers.set('Authorization', `Bearer ${accessToken}`);
    const visitorId = getVisitorId();
    if (visitorId) headers.set('X-Visitor-Id', visitorId);
    return fetch('/api/v1/rag/chat/stream', {
      method: 'POST',
      body: JSON.stringify(data),
      headers,
      credentials: 'include',
      signal,
    });
  };

  const initial = await request(useAuthStore.getState().accessToken);
  // Fetch 不会经过 Axios 的刷新拦截器；仅对未中止的 401 刷新一次后重试。
  if (initial.status !== 401 || signal?.aborted) return initial;
  try {
    const accessToken = await refreshAccessToken();
    useAuthStore.setState({ accessToken, isInitialized: true });
    const retried = await request(accessToken);
    if (retried.status !== 401 || signal?.aborted) return retried;

    // 刷新成功后仍被拒绝，说明会话在重试窗口中失效或服务端已撤销它。和
    // Axios 拦截器一致地清理内存认证状态，避免继续展示私有历史。
    const { logout, onSessionExpired } = useAuthStore.getState();
    logout();
    onSessionExpired?.();
    return retried;
  } catch {
    // 与 Axios 拦截器的 401 策略保持一致：Cookie 刷新失败说明已无法恢复登录态，
    // 不得让页面继续持有旧用户的会话数据或旧 access token。
    const { logout, onSessionExpired } = useAuthStore.getState();
    logout();
    onSessionExpired?.();
    return initial;
  }
};

const responseErrorMessage = async (response: Response): Promise<string> => {
  try {
    const payload = await response.json() as { message?: unknown };
    if (typeof payload.message === 'string' && payload.message.trim()) return payload.message;
  } catch {
    // 服务端可能在代理/网关层返回非 JSON，保留状态文本作为后备。
  }
  return response.statusText || `RAG request failed (${response.status})`;
};
