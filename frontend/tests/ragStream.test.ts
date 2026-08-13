import assert from 'node:assert/strict';
import test from 'node:test';

import { createRAGStreamDecoder, parseRAGStreamBlock, type RAGParsedStreamEvent } from '../src/utils/ragStream.ts';

test('SSE 解析支持 event 字段、JSON delta 与中文内容', () => {
  assert.deepEqual(
    parseRAGStreamBlock('event: delta\ndata: {"delta":"你好，世界"}\n\n'),
    { type: 'delta', delta: '你好，世界' },
  );
});

test('SSE 解析支持来源、会话元数据与结束事件', () => {
  assert.deepEqual(
    parseRAGStreamBlock('event: sources\ndata: {"sources":[{"articleId":7,"title":"RAG 入门","snippet":"检索增强"}]}\n\n'),
    {
      type: 'sources',
      sources: [{ articleId: 7, title: 'RAG 入门', snippet: '检索增强', url: undefined }],
    },
  );
  assert.deepEqual(
    parseRAGStreamBlock('data: {"type":"done","sessionId":"s-1","messageId":"m-1"}\n\n'),
    { type: 'done', sessionId: 's-1', messageId: 'm-1' },
  );
});

test('SSE 的纯文本与不完整事件安全降级', () => {
  assert.deepEqual(parseRAGStreamBlock('data: text fragment\n\n'), { type: 'delta', delta: 'text fragment' });
  assert.equal(parseRAGStreamBlock(': keepalive\n\n'), null);
  assert.deepEqual(parseRAGStreamBlock('event: unknown\ndata: {"content":"fallback"}\n\n'), {
    type: 'delta',
    delta: 'fallback',
  });
});

test('SSE 保留纯文本 delta 的有效前导空白', () => {
  assert.deepEqual(parseRAGStreamBlock('data:  leading space\n\n'), { type: 'delta', delta: ' leading space' });
});

test('SSE 解码器可跨 UTF-8 网络分块还原中文事件', () => {
  const events: RAGParsedStreamEvent[] = [];
  const decoder = createRAGStreamDecoder((event) => events.push(event));
  const bytes = new TextEncoder().encode('event: delta\ndata: {"delta":"你好"}\n\n');
  // 第二个汉字的 UTF-8 字节落在两个网络分块之间。
  const splitAt = bytes.length - 4;
  decoder.push(bytes.slice(0, splitAt));
  decoder.push(bytes.slice(splitAt));
  decoder.finish();

  assert.deepEqual(events, [{ type: 'delta', delta: '你好' }]);
});

test('SSE 解码器可跨 CRLF 边界还原多个事件，并在 finish 时处理尾部事件', () => {
  const events: RAGParsedStreamEvent[] = [];
  const decoder = createRAGStreamDecoder((event) => events.push(event));
  const encoder = new TextEncoder();

  decoder.push(encoder.encode('event: meta\r\ndata: {"sessionId":"s-1"}\r'));
  decoder.push(encoder.encode('\n\r\nevent: delta\r\ndata: {"delta":"尾部"}'));
  decoder.finish();

  assert.deepEqual(events, [
    { type: 'meta', sessionId: 's-1' },
    { type: 'delta', delta: '尾部' },
  ]);
});

test('SSE 仅忽略 data 冒号后的一个空格，并拼接多行 data', () => {
  assert.deepEqual(
    parseRAGStreamBlock('event: delta\ndata: 第一行\ndata:  第二行\n\n'),
    { type: 'delta', delta: '第一行\n 第二行' },
  );
});
