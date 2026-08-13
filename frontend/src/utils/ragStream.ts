import type { RAGChatSource } from '../types';

export type RAGParsedStreamEvent =
  | { type: 'meta'; sessionId?: string }
  | { type: 'sources'; sources: RAGChatSource[] }
  | { type: 'delta'; delta: string }
  | { type: 'done'; sessionId?: string; messageId?: string }
  | { type: 'error'; message: string };

export type RAGStreamDecoder = {
  push: (chunk: Uint8Array) => void;
  finish: () => void;
};

/**
 * 解析单个 SSE 事件块。服务端可把类型放在 `event:`、`type` 或 `event` 字段中；
 * 纯文本 data 则安全降级为 delta，便于兼容不同的 go-zero 流式写法。
 */
export const parseRAGStreamBlock = (block: string): RAGParsedStreamEvent | null => {
  const lines = block.split(/\r?\n/);
  const dataLines: string[] = [];
  let eventName = '';
  for (const line of lines) {
    if (line.startsWith('event:')) {
      eventName = line.slice('event:'.length).trim();
    } else if (line.startsWith('data:')) {
      // SSE 规范只忽略 `:` 后的一个可选空格；不能使用 trimStart，
      // 否则纯文本 delta 开头的空白会被意外吞掉。
      const value = line.slice('data:'.length);
      dataLines.push(value.startsWith(' ') ? value.slice(1) : value);
    }
  }
  if (dataLines.length === 0) return null;
  return toRAGStreamEvent(eventName, dataLines.join('\n'));
};

export const toRAGStreamEvent = (eventName: string, raw: string): RAGParsedStreamEvent => {
  let payload: Record<string, unknown> = {};
  try {
    const parsed: unknown = JSON.parse(raw);
    if (typeof parsed === 'object' && parsed !== null) payload = parsed as Record<string, unknown>;
  } catch {
    return { type: 'delta', delta: raw };
  }

  const type = normalizeEventType(eventName || stringValue(payload.type) || stringValue(payload.event));
  switch (type) {
    case 'meta':
      return { type, sessionId: stringValue(payload.sessionId) };
    case 'sources':
      return { type, sources: normalizeRAGSources(payload.sources) };
    case 'done':
      return { type, sessionId: stringValue(payload.sessionId), messageId: stringValue(payload.messageId) };
    case 'error':
      return { type, message: stringValue(payload.message) || 'RAG stream failed' };
    case 'delta':
    default:
      return { type: 'delta', delta: stringValue(payload.delta) || stringValue(payload.content) || stringValue(payload.text) };
  }
};

const normalizeEventType = (value: string): RAGParsedStreamEvent['type'] => {
  switch (value) {
    case 'meta':
    case 'sources':
    case 'done':
    case 'error':
    case 'delta':
      return value;
    default:
      return 'delta';
  }
};

const stringValue = (value: unknown): string => typeof value === 'string' ? value : '';

export const normalizeRAGSources = (value: unknown): RAGChatSource[] => {
  if (!Array.isArray(value)) return [];
  return value.flatMap((item): RAGChatSource[] => {
    if (typeof item !== 'object' || item === null) return [];
    const source = item as Record<string, unknown>;
    const articleId = Number(source.articleId);
    const title = stringValue(source.title);
    if (!Number.isInteger(articleId) || articleId <= 0 || !title) return [];
    return [{
      articleId,
      title,
      url: stringValue(source.url) || undefined,
      snippet: stringValue(source.snippet) || undefined,
    }];
  });
};

/**
 * 将任意网络分块还原为完整 SSE 事件。TextDecoder 的 stream 模式会保留未完成的 UTF-8
 * 字节序列，因此中文恰好跨分块时也不会产生乱码。
 */
export const createRAGStreamDecoder = (
  onEvent: (event: RAGParsedStreamEvent) => void,
): RAGStreamDecoder => {
  const decoder = new TextDecoder();
  let buffer = '';

  const consume = (text: string) => {
    if (!text) return;
    buffer += text;
    const blocks = buffer.split(/\r?\n\r?\n/);
    buffer = blocks.pop() ?? '';
    blocks.forEach((block) => {
      const event = parseRAGStreamBlock(block);
      if (event) onEvent(event);
    });
  };

  return {
    push: (chunk) => consume(decoder.decode(chunk, { stream: true })),
    finish: () => {
      consume(decoder.decode());
      if (!buffer.trim()) return;
      const event = parseRAGStreamBlock(buffer);
      buffer = '';
      if (event) onEvent(event);
    },
  };
};
