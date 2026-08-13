import React, { useCallback, useEffect, useMemo, useRef, useState } from 'react';
import { Link } from 'react-router-dom';
import Button from '../components/ui/Button';
import EmptyState from '../components/ui/EmptyState';
import InlineNotice from '../components/InlineNotice';
import MarkdownRenderer from '../components/MarkdownRenderer';
import PagePendingState from '../components/RoutePending';
import {
  deleteRAGSession,
  getRAGSession,
  getRAGSessions,
  streamRAGChat,
  type RAGStreamEvent,
} from '../api/rag';
import { useConfirm } from '../hooks/useConfirm';
import { getDateLocale, translate } from '../i18n';
import { useAuthStore } from '../store/auth';
import { usePreferenceStore } from '../store/preferences';
import { useSiteSettingsStore } from '../store/siteSettings';
import type { RAGChatMessage, RAGChatSession, RAGChatSource } from '../types';
import { getErrorMessage } from '../utils/error';
import { useSEO } from '../utils/seo';

const QUESTION_LIMIT = 4_000;
// 距底部小于该像素数时视为“正在跟随底部”，流式更新才自动滚动。
const SCROLL_STICK_THRESHOLD = 96;

const Ask: React.FC = () => {
  const language = usePreferenceStore((state) => state.language);
  const { user, isInitialized } = useAuthStore((state) => ({
    user: state.user,
    isInitialized: state.isInitialized,
  }));
  const accessLevel = useSiteSettingsStore((state) => state.ragChatAccessLevel);
  const [question, setQuestion] = useState('');
  const [messages, setMessages] = useState<RAGChatMessage[]>([]);
  const [sessions, setSessions] = useState<RAGChatSession[]>([]);
  const [sessionId, setSessionId] = useState<string | undefined>();
  const [sessionsLoading, setSessionsLoading] = useState(false);
  const [sessionLoading, setSessionLoading] = useState(false);
  const [streaming, setStreaming] = useState(false);
  const [streamingAssistantId, setStreamingAssistantId] = useState<string>();
  const [error, setError] = useState('');
  const [historyError, setHistoryError] = useState('');
  const controllerRef = useRef<AbortController | null>(null);
  const sessionRequestRef = useRef<AbortController | null>(null);
  const activeUserIDRef = useRef<number | undefined>(user?.id);
  // SSE 的 meta/done 事件可能在 React 状态提交前到达；保留即时值以便首次提问能写入会话列表。
  const sessionIdRef = useRef<string | undefined>();
  const scrollContainerRef = useRef<HTMLDivElement | null>(null);
  // 用户上翻阅读时暂停自动跟随，滚回底部附近后恢复，避免生成内容时被强制拉到底部。
  const stickToBottomRef = useRef(true);
  const confirm = useConfirm();
  const t = useCallback((key: Parameters<typeof translate>[1]) => translate(language, key), [language]);
  const canAccess = isChatAccessAllowed(accessLevel, user?.role);
  // 即使 guest 可访问，也要先恢复登录态：否则已登录用户在刷新页后的首次提问会被误作游客会话。
  const shouldWaitForAuth = !isInitialized;

  useSEO(t('ragChat.title'));

  useEffect(() => {
    if (!user || !canAccess) {
      setSessions([]);
      return undefined;
    }
    const controller = new AbortController();
    setSessionsLoading(true);
    setHistoryError('');
    getRAGSessions(controller.signal)
      .then((response) => {
        if (!controller.signal.aborted) {
          setSessions(normalizeSessionList(response.data));
        }
      })
      .catch((requestError) => {
        if (!controller.signal.aborted) {
          setHistoryError(getErrorMessage(requestError, t('ragChat.loadSessionsError')));
        }
      })
      .finally(() => {
        if (!controller.signal.aborted) setSessionsLoading(false);
      });
    return () => controller.abort();
  }, [canAccess, t, user]);

  useEffect(() => {
    // 会话属于登录用户。登出、账户切换或访问级别收紧时，必须取消当前请求并清除
    // 仅存在于前端内存中的内容，避免在同一浏览器窗口泄露上一账户的私有历史。
    const nextUserID = user?.id;
    if (activeUserIDRef.current === nextUserID && canAccess) return;
    activeUserIDRef.current = nextUserID;
    controllerRef.current?.abort();
    controllerRef.current = null;
    sessionRequestRef.current?.abort();
    sessionRequestRef.current = null;
    sessionIdRef.current = undefined;
    setSessionId(undefined);
    setMessages([]);
    setSessions([]);
    setSessionLoading(false);
    setStreaming(false);
    setStreamingAssistantId(undefined);
    setError('');
    setHistoryError('');
  }, [canAccess, user?.id]);

  useEffect(() => {
    const container = scrollContainerRef.current;
    if (!container || !stickToBottomRef.current) return;
    container.scrollTop = container.scrollHeight;
  }, [messages]);

  const handleScroll = () => {
    const container = scrollContainerRef.current;
    if (!container) return;
    stickToBottomRef.current = container.scrollHeight - container.scrollTop - container.clientHeight < SCROLL_STICK_THRESHOLD;
  };

  useEffect(() => () => {
    controllerRef.current?.abort();
    sessionRequestRef.current?.abort();
  }, []);

  const sessionItems = useMemo(
    () => [...sessions].sort((a, b) => Date.parse(b.updatedAt) - Date.parse(a.updatedAt)),
    [sessions],
  );

  const startNewChat = () => {
    controllerRef.current?.abort();
    controllerRef.current = null;
    sessionRequestRef.current?.abort();
    sessionIdRef.current = undefined;
    setSessionId(undefined);
    setMessages([]);
    setQuestion('');
    setError('');
    setSessionLoading(false);
    setStreamingAssistantId(undefined);
  };

  const loadSession = async (nextSessionId: string) => {
    if (streaming || nextSessionId === sessionId) return;
    sessionRequestRef.current?.abort();
    const controller = new AbortController();
    sessionRequestRef.current = controller;
    setSessionLoading(true);
    setError('');
    try {
      const response = await getRAGSession(nextSessionId, controller.signal);
      if (controller.signal.aborted || sessionRequestRef.current !== controller) return;
      const session = normalizeSession(response.data);
      sessionIdRef.current = session.id;
      setSessionId(session.id);
      stickToBottomRef.current = true;
      setMessages(session.messages ?? []);
    } catch (requestError) {
      if (!controller.signal.aborted) {
        setError(getErrorMessage(requestError, t('ragChat.sessionLoadError')));
      }
    } finally {
      if (sessionRequestRef.current === controller) {
        sessionRequestRef.current = null;
        if (!controller.signal.aborted) setSessionLoading(false);
      }
    }
  };

  const removeSession = async (event: React.MouseEvent<HTMLButtonElement>, target: RAGChatSession) => {
    event.stopPropagation();
    if (streaming) return;
    const accepted = await confirm({
      title: t('ragChat.deleteSession'),
      description: t('ragChat.deleteConfirm'),
      confirmLabel: t('common.delete'),
      cancelLabel: t('common.cancel'),
      tone: 'danger',
    });
    if (!accepted) return;

    try {
      await deleteRAGSession(target.id);
      setSessions((current) => current.filter((session) => session.id !== target.id));
      if (target.id === sessionId) startNewChat();
    } catch (requestError) {
      setHistoryError(getErrorMessage(requestError, t('ragChat.deleteError')));
    }
  };

  /**
   * 执行一次流式问答并写入指定 assistant 消息。提交新问题与重试共用，
   * 保证 Abort、会话摘要更新、错误标记等行为完全一致。
   */
  const runStream = async (question: string, assistantId: string) => {
    const controller = new AbortController();
    controllerRef.current = controller;
    setError('');
    setStreaming(true);
    setStreamingAssistantId(assistantId);

    try {
      await streamRAGChat(
        { question, sessionId: sessionIdRef.current },
        {
          onEvent: (streamEvent) => {
            // Abort 后仍可能有已缓冲事件送达；不允许旧请求污染新会话。
            if (controller.signal.aborted || controllerRef.current !== controller) return;
            handleStreamEvent(
              streamEvent,
              assistantId,
              setMessages,
              (nextSessionId) => {
                sessionIdRef.current = nextSessionId;
                setSessionId(nextSessionId);
              },
              (message) => setError(message),
            );
          },
        },
        controller.signal,
      );
      if (user && !controller.signal.aborted) {
        // SSE 完成后只更新列表摘要，避免再取整段消息打断当前流式内容。
        setSessions((current) => upsertLocalSession(current, sessionIdRef.current, question));
      }
    } catch (requestError) {
      if (!controller.signal.aborted) {
        setMessages((current) => markAssistantIncomplete(current, assistantId));
        setError(getErrorMessage(requestError, t('ragChat.sendError')));
      }
    } finally {
      // 停止后用户可以立刻发起下一轮；旧请求结束时不能反向关闭新请求的流式状态。
      if (controllerRef.current === controller) {
        controllerRef.current = null;
        setStreaming(false);
        setStreamingAssistantId(undefined);
      }
    }
  };

  const submitQuestion = async (event: React.FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    const normalizedQuestion = question.trim();
    if (!normalizedQuestion) {
      setError(t('ragChat.questionRequired'));
      return;
    }
    if (streaming || normalizedQuestion.length > QUESTION_LIMIT) return;

    const now = new Date().toISOString();
    const localUserMessage: RAGChatMessage = {
      id: `local-user-${createLocalID()}`,
      sessionId,
      role: 'user',
      content: normalizedQuestion,
      createdAt: now,
    };
    const localAssistantId = `local-assistant-${createLocalID()}`;
    const localAssistantMessage: RAGChatMessage = {
      id: localAssistantId,
      sessionId,
      role: 'assistant',
      content: '',
      sources: [],
      createdAt: now,
    };
    // 新一轮请求前清除上一次的即时会话 ID，避免停止首轮生成后把下一轮写入孤儿会话。
    sessionIdRef.current = sessionId;
    stickToBottomRef.current = true;
    setMessages((current) => [...current, localUserMessage, localAssistantMessage]);
    setQuestion('');
    await runStream(normalizedQuestion, localAssistantId);
  };

  /** 重新生成失败或被中断的回答：找回对应的用户问题并重新发起流式请求。 */
  const retryAnswer = (assistantId: string) => {
    if (streaming) return;
    const index = messages.findIndex((message) => message.id === assistantId);
    if (index < 0) return;
    const userMessage = [...messages.slice(0, index)].reverse().find((message) => message.role === 'user');
    if (!userMessage) return;
    const question = userMessage.content;
    // 清空旧回答与引用，等待新的流式结果；中断/失败轮次未保存到会话，不会重复。
    setMessages((current) => current.map((message) => (
      message.id === assistantId
        ? { ...message, content: '', sources: [], sourcesUpdated: false, incomplete: false }
        : message
    )));
    stickToBottomRef.current = true;
    void runStream(question, assistantId);
  };

  const stopStreaming = () => {
    const controller = controllerRef.current;
    if (!controller) return;
    controller.abort();
    controllerRef.current = null;
    setMessages((current) => markAssistantIncomplete(current, streamingAssistantId));
    setError(t('ragChat.interrupted'));
    setStreaming(false);
    setStreamingAssistantId(undefined);
  };

  if (shouldWaitForAuth) {
    return <PagePendingState />;
  }

  if (!canAccess) {
    const accessMessage = accessLevel === 'user'
      ? t('ragChat.signInRequired')
      : accessLevel === 'editor'
        ? t('ragChat.editorRequired')
        : t('ragChat.accessDenied');
    return (
      <div className="editorial-container w-full max-w-3xl">
        <InlineNotice
          message={accessMessage}
          tone="warning"
          icon
          action={!user && accessLevel !== 'guest' ? (
            <Link to="/login" className="text-xs font-bold tracking-widest text-ochre hover:text-ink">
              {t('ragChat.signIn')}
            </Link>
          ) : undefined}
        />
      </div>
    );
  }

  return (
    <div className="editorial-container w-full">
      <section className="relative overflow-hidden rounded-xl bg-surface-soft px-5 py-4 md:px-6 md:py-5">
        <div className="absolute left-0 top-3 bottom-3 w-px bg-ochre opacity-60" />
        <div className="flex flex-col gap-1.5 md:flex-row md:items-baseline md:justify-between md:gap-6">
          <div className="flex items-baseline gap-3">
            <p className="text-[0.65rem] tracking-[0.32em] text-ochre">{t('ragChat.kicker')}</p>
            <h1 className="font-display text-2xl leading-tight text-ink md:text-3xl">{t('ragChat.title')}</h1>
          </div>
          <p className="text-xs leading-relaxed tracking-wide text-ink-light md:max-w-md md:text-right">{t('ragChat.subtitle')}</p>
        </div>
      </section>

      <div className="mt-8 grid gap-6 xl:grid-cols-[16rem_minmax(0,1fr)]">
        {user && (
          <aside className="rounded-xl border border-hairline bg-paper p-4 xl:sticky xl:top-24 xl:h-fit">
            <div className="flex items-center justify-between gap-3 border-b border-hairline pb-3">
              <h2 className="text-xs font-bold tracking-widest text-ink">{t('ragChat.history')}</h2>
              <Button size="sm" variant="subtle" disabled={streaming} onClick={startNewChat}>
                {t('ragChat.newChat')}
              </Button>
            </div>
            <InlineNotice message={historyError} className="mt-3" />
            {sessionsLoading ? (
              <PagePendingState variant="inline" label={t('ragChat.historyLoading')} />
            ) : sessionItems.length === 0 ? (
              <p className="py-5 text-xs leading-relaxed text-ink-light">{t('ragChat.historyEmpty')}</p>
            ) : (
              <ul className="mt-3 max-h-72 space-y-1 overflow-y-auto xl:max-h-[calc(100vh-17rem)]">
                {sessionItems.map((item) => (
                  <li key={item.id}>
                    <div className={`group flex items-center gap-1 rounded-md ${item.id === sessionId ? 'bg-surface-soft' : 'hover:bg-surface-soft'}`}>
                      <button
                        type="button"
                        onClick={() => void loadSession(item.id)}
                        disabled={streaming || sessionLoading}
                        className="min-w-0 flex-1 px-3 py-2.5 text-left text-sm text-ink disabled:cursor-not-allowed disabled:opacity-60"
                        title={item.title}
                      >
                        <span className="block truncate">{item.title || t('ragChat.newChat')}</span>
                        <span className="mt-1 block text-[0.65rem] tracking-wide text-muted">
                          {formatSessionDate(item.updatedAt, language)}
                        </span>
                      </button>
                      <button
                        type="button"
                        aria-label={`${t('ragChat.deleteSession')}: ${item.title}`}
                        disabled={streaming}
                        onClick={(event) => void removeSession(event, item)}
                        className="mr-1 flex h-9 w-9 shrink-0 items-center justify-center rounded text-muted opacity-80 transition-colors hover:bg-paper hover:text-ember disabled:cursor-not-allowed disabled:opacity-40"
                      >
                        ×
                      </button>
                    </div>
                  </li>
                ))}
              </ul>
            )}
          </aside>
        )}

        <section className="min-w-0 rounded-xl border border-hairline bg-paper">
          <div
            ref={scrollContainerRef}
            onScroll={handleScroll}
            className="max-h-[min(62vh,54rem)] min-h-[24rem] overflow-y-auto p-5 md:p-7"
          >
            {sessionLoading ? (
              <PagePendingState variant="inline" label={t('ragChat.historyLoading')} />
            ) : messages.length === 0 ? (
              <EmptyState
                illustration="ink-drop"
                title={t('ragChat.emptyTitle')}
                description={t('ragChat.emptyDesc')}
                className="min-h-[22rem] bg-transparent"
              />
            ) : (
              <div className="space-y-7">
                {messages.map((message) => (
                  <ChatMessageCard
                    key={message.id}
                    message={message}
                    streaming={message.id === streamingAssistantId}
                    t={t}
                    onRetry={retryAnswer}
                  />
                ))}
              </div>
            )}
          </div>
          <div className="border-t border-hairline bg-surface-soft/50 p-4 md:p-5">
            <InlineNotice message={error} className="mb-3" />
            <form onSubmit={(event) => void submitQuestion(event)} className="flex flex-col gap-3">
              <textarea
                value={question}
                maxLength={QUESTION_LIMIT}
                rows={3}
                disabled={streaming}
                onChange={(event) => setQuestion(event.target.value)}
                onKeyDown={(event) => {
                  if (event.key === 'Enter' && (event.ctrlKey || event.metaKey)) {
                    event.preventDefault();
                    event.currentTarget.form?.requestSubmit();
                  }
                }}
                placeholder={t('ragChat.askPlaceholder')}
                aria-label={t('ragChat.askPlaceholder')}
                className="w-full resize-y border border-mountain-grey bg-paper px-3 py-3 text-sm leading-relaxed text-ink outline-hidden placeholder:text-muted focus:border-ochre disabled:cursor-not-allowed disabled:opacity-60"
              />
              <div className="flex flex-wrap items-center justify-between gap-3">
                <span className="text-xs text-muted">{question.length}/{QUESTION_LIMIT} · Ctrl/⌘ + Enter</span>
                {streaming ? (
                  <Button type="button" variant="ghost" onClick={stopStreaming}>
                    {t('ragChat.stop')}
                  </Button>
                ) : (
                  <Button type="submit" variant="primary" disabled={!question.trim()}>
                    {t('ragChat.send')}
                  </Button>
                )}
              </div>
            </form>
          </div>
        </section>
      </div>
    </div>
  );
};

const ChatMessageCard = ({
  message,
  streaming,
  t,
  onRetry,
}: {
  message: RAGChatMessage;
  streaming: boolean;
  t: (key: Parameters<typeof translate>[1]) => string;
  onRetry?: (assistantId: string) => void;
}) => {
  const isAssistant = message.role === 'assistant';
  const [copied, setCopied] = useState(false);
  const copyTimerRef = useRef<number | undefined>(undefined);

  useEffect(() => () => {
    if (copyTimerRef.current) {
      window.clearTimeout(copyTimerRef.current);
    }
  }, []);

  const copyAnswer = async () => {
    try {
      if (navigator.clipboard?.writeText) {
        try {
          await navigator.clipboard.writeText(message.content);
        } catch {
          // 本地开发环境或非安全上下文可能拒绝异步剪贴板，继续使用兼容路径。
          copyWithLegacyApi(message.content);
        }
      } else {
        copyWithLegacyApi(message.content);
      }
      setCopied(true);
      if (copyTimerRef.current) {
        window.clearTimeout(copyTimerRef.current);
      }
      copyTimerRef.current = window.setTimeout(() => setCopied(false), 1600);
    } catch {
      setCopied(false);
    }
  };

  // 流式开始时参考文章先于回答到达；回答内容为空时暂不展示，
  // 避免“切片先出现、页面直接跳到底部”的错位观感。
  const showSources = (message.sources?.length ?? 0) > 0 && !(streaming && !message.content);

  return (
    <article className={isAssistant ? '' : 'ml-auto max-w-3xl'}>
      <p className={`mb-2 text-xs font-bold tracking-widest ${isAssistant ? 'text-ochre' : 'text-ink-light'}`}>
        {isAssistant ? t('ragChat.assistant') : t('ragChat.you')}
      </p>
      {message.hiddenAt ? (
        <InlineNotice message={t('ragChat.unavailable')} tone="warning" icon />
      ) : isAssistant ? (
        <div className="rounded-lg border border-hairline bg-surface-soft/50 px-4 py-4 md:px-5">
          {message.content ? (
            streaming ? (
              <div className="whitespace-pre-wrap text-sm leading-7 text-ink">{message.content}</div>
            ) : (
              <MarkdownRenderer content={message.content} className="text-sm leading-7" />
            )
          ) : streaming ? (
            <p className="animate-pulse text-sm text-ink-light">{t('ragChat.generating')}</p>
          ) : null}
          {message.sourcesUpdated && <InlineNotice message={t('ragChat.sourcesUpdated')} tone="warning" icon className="mt-4" />}
          {message.incomplete && <InlineNotice message={t('ragChat.interrupted')} tone="warning" icon className="mt-4" />}
          {showSources && (
            <SourceList sources={message.sources ?? []} hideSnippets={Boolean(message.sourcesUpdated)} t={t} />
          )}
          {!streaming && (message.content || message.incomplete) && (
            <div className="mt-4 flex flex-wrap items-center gap-2 border-t border-hairline pt-3">
              {message.incomplete && (
                <Button size="sm" variant="subtle" onClick={() => onRetry?.(message.id)}>
                  {t('ragChat.retry')}
                </Button>
              )}
              {message.content && (
                <Button size="sm" variant="subtle" onClick={() => void copyAnswer()}>
                  {copied ? t('ragChat.copied') : t('ragChat.copy')}
                </Button>
              )}
            </div>
          )}
        </div>
      ) : (
        <div>
          <div className="rounded-lg bg-ink px-4 py-3 text-sm leading-7 text-on-dark">
            <div className="whitespace-pre-wrap">{message.content}</div>
          </div>
          {!streaming && message.content && (
            <div className="mt-1.5 flex justify-end">
              <button
                type="button"
                onClick={() => void copyAnswer()}
                className="text-xs text-muted transition-colors hover:text-ink"
              >
                {copied ? t('ragChat.copied') : t('ragChat.copy')}
              </button>
            </div>
          )}
        </div>
      )}
    </article>
  );
};

const SourceList = ({
  sources,
  hideSnippets,
  t,
}: {
  sources: RAGChatSource[];
  hideSnippets?: boolean;
  t: (key: Parameters<typeof translate>[1]) => string;
}) => (
  <div className="mt-5 border-t border-hairline pt-4">
    <p className="text-xs font-bold tracking-widest text-ink-light">{t('ragChat.sources')}</p>
    <ul className="mt-3 grid gap-2">
      {sources.map((source) => (
        <li key={`${source.articleId}-${source.title}`} className="rounded-md border border-hairline bg-paper px-3 py-2.5">
          <Link to={source.url || `/article/${source.articleId}`} className="text-sm font-medium text-ochre hover:text-ink">
            {source.title}
          </Link>
          {!hideSnippets && source.snippet && <p className="mt-1.5 line-clamp-2 text-xs leading-relaxed text-ink-light">{source.snippet}</p>}
        </li>
      ))}
    </ul>
  </div>
);

const handleStreamEvent = (
  event: RAGStreamEvent,
  assistantId: string,
  setMessages: React.Dispatch<React.SetStateAction<RAGChatMessage[]>>,
  setSessionId: (sessionId: string) => void,
  setError: React.Dispatch<React.SetStateAction<string>>,
) => {
  if (event.type === 'meta' && event.sessionId) {
    setSessionId(event.sessionId);
    return;
  }
  if (event.type === 'done' && event.sessionId) {
    setSessionId(event.sessionId);
  }
  if (event.type === 'error') {
    setError(event.message);
    return;
  }
  if (event.type !== 'delta' && event.type !== 'sources') return;
  setMessages((current) => current.map((message) => {
    if (message.id !== assistantId) return message;
    if (event.type === 'sources') return { ...message, sources: event.sources };
    return { ...message, content: `${message.content}${event.delta}` };
  }));
};

const markAssistantIncomplete = (messages: RAGChatMessage[], assistantID: string | undefined): RAGChatMessage[] => {
  if (!assistantID) return messages;
  return messages.map((message) => (
    message.id === assistantID ? { ...message, incomplete: true } : message
  ));
};

const isChatAccessAllowed = (accessLevel: string, role?: string): boolean => {
  if (accessLevel === 'guest') return true;
  if (accessLevel === 'user') return Boolean(role);
  return role === 'editor' || role === 'admin';
};

const normalizeSessionList = (value: unknown): RAGChatSession[] => {
  const items = Array.isArray(value)
    ? value
    : typeof value === 'object' && value !== null && Array.isArray((value as { items?: unknown }).items)
      ? (value as { items: unknown[] }).items
      : [];
  return items.flatMap((item) => {
    const session = normalizeSession(item);
    return session.id ? [session] : [];
  });
};

const normalizeSession = (value: unknown): RAGChatSession => {
  const root = typeof value === 'object' && value !== null ? value as Record<string, unknown> : {};
  // 兼容 `{ session, messages }` 和直接 session 两种稳定、无歧义的响应封装。
  const embedded = typeof root.session === 'object' && root.session !== null
    ? root.session as Record<string, unknown>
    : root;
  const source: Record<string, unknown> = { ...embedded, messages: root.messages ?? embedded.messages };
  const rawMessages = Array.isArray(source.messages) ? source.messages : [];
  return {
    id: String(source.id ?? ''),
    title: typeof source.title === 'string' ? source.title : '',
    sourceEpoch: typeof source.sourceEpoch === 'number' ? source.sourceEpoch : undefined,
    expiresAt: typeof source.expiresAt === 'string' ? source.expiresAt : undefined,
    createdAt: typeof source.createdAt === 'string' ? source.createdAt : '',
    updatedAt: typeof source.updatedAt === 'string' ? source.updatedAt : '',
    messages: rawMessages.flatMap(normalizeMessage),
  };
};

const normalizeMessage = (value: unknown): RAGChatMessage[] => {
  if (typeof value !== 'object' || value === null) return [];
  const source = value as Record<string, unknown>;
  if (source.role !== 'user' && source.role !== 'assistant') return [];
  return [{
    id: String(source.id ?? createLocalID()),
    sessionId: typeof source.sessionId === 'string' ? source.sessionId : undefined,
    role: source.role,
    content: typeof source.content === 'string' ? source.content : '',
    sources: normalizeSources(source.sources),
    sourcesUpdated: source.sourcesUpdated === true,
    incomplete: source.incomplete === true,
    hiddenAt: typeof source.hiddenAt === 'string' ? source.hiddenAt : undefined,
    createdAt: typeof source.createdAt === 'string' ? source.createdAt : '',
  }];
};

const normalizeSources = (value: unknown): RAGChatSource[] => {
  if (typeof value === 'string') {
    try {
      return normalizeSources(JSON.parse(value));
    } catch {
      return [];
    }
  }
  if (!Array.isArray(value)) return [];
  return value.flatMap((item): RAGChatSource[] => {
    if (typeof item !== 'object' || item === null) return [];
    const source = item as Record<string, unknown>;
    const articleId = Number(source.articleId);
    const title = typeof source.title === 'string' ? source.title : '';
    if (!Number.isInteger(articleId) || articleId < 1 || !title) return [];
    return [{
      articleId,
      title,
      url: typeof source.url === 'string' ? source.url : undefined,
      snippet: typeof source.snippet === 'string' ? source.snippet : undefined,
    }];
  });
};

const upsertLocalSession = (
  sessions: RAGChatSession[],
  id: string | undefined,
  question: string,
): RAGChatSession[] => {
  if (!id) return sessions;
  const title = question.length > 48 ? `${question.slice(0, 48)}…` : question;
  const updatedAt = new Date().toISOString();
  const existing = sessions.find((session) => session.id === id);
  if (existing) {
    return sessions.map((session) => session.id === id ? { ...session, title: session.title || title, updatedAt } : session);
  }
  return [{ id, title, createdAt: updatedAt, updatedAt }, ...sessions];
};

const formatSessionDate = (value: string, language: 'zh' | 'en'): string => {
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) return '';
  return new Intl.DateTimeFormat(getDateLocale(language), { month: 'short', day: 'numeric' }).format(date);
};

const createLocalID = (): string => {
  if (typeof crypto !== 'undefined' && typeof crypto.randomUUID === 'function') {
    return crypto.randomUUID();
  }
  return `${Date.now()}-${Math.random().toString(36).slice(2)}`;
};

/** 非安全上下文下 navigator.clipboard 不可用时的复制兜底。 */
const copyWithLegacyApi = (text: string) => {
  const input = document.createElement('textarea');
  input.value = text;
  input.setAttribute('readonly', '');
  input.style.position = 'fixed';
  input.style.top = '0';
  input.style.left = '-9999px';
  input.style.opacity = '0';
  input.style.pointerEvents = 'none';
  document.body.appendChild(input);
  try {
    input.focus();
    input.select();
    if (!document.execCommand('copy')) {
      throw new Error('copy failed');
    }
  } finally {
    input.remove();
  }
};

export default Ask;
