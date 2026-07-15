import assert from 'node:assert/strict';
import test from 'node:test';
import {
  dateToUTCBoundary,
  formatLogResource,
  getClientSummary,
  getLogEventPresentation,
  parseLogMetadata,
} from '../src/pages/admin/logPresentation.ts';

test('日志事件和资源按语言展示并兼容未知值', () => {
  assert.equal(getLogEventPresentation('article.created', 'zh').label, '创建文章');
  assert.equal(getLogEventPresentation('article.created', 'en').label, 'Article created');
  assert.equal(getLogEventPresentation('future.event', 'zh').label, 'future.event');
  assert.equal(formatLogResource({ resourceType: 'article', resourceId: 12 }, 'zh'), '文章 #12');
  assert.equal(formatLogResource({ resourceType: 'custom', resourceId: undefined }, 'en'), 'custom');
});

test('日志 metadata 支持对象解析和旧格式原文兜底', () => {
  assert.deepEqual(parseLogMetadata('{"status":"published","versionNo":2}').entries, [
    { key: 'status', value: 'published' },
    { key: 'versionNo', value: '2' },
  ]);
  assert.deepEqual(parseLogMetadata('').entries, []);
  assert.equal(parseLogMetadata('legacy=value').invalid, true);
  assert.equal(parseLogMetadata('[1,2]').invalid, true);
});

test('客户端摘要识别常见浏览器并处理空值', () => {
  const chrome = 'Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 Chrome/126.0.0.0 Safari/537.36';
  assert.equal(getClientSummary(chrome, 'zh'), 'Chrome 126.0.0.0 · Windows');
  assert.equal(getClientSummary('', 'en'), 'Unknown client');
});

test('日期边界使用本地自然日且结束日期为次日零点', () => {
  const startValue = dateToUTCBoundary('2026-07-15', false);
  const endValue = dateToUTCBoundary('2026-07-15', true);
  assert.ok(startValue);
  assert.ok(endValue);
  const start = new Date(startValue);
  const end = new Date(endValue);
  assert.deepEqual([start.getFullYear(), start.getMonth(), start.getDate()], [2026, 6, 15]);
  assert.deepEqual([end.getFullYear(), end.getMonth(), end.getDate()], [2026, 6, 16]);
  assert.equal(dateToUTCBoundary('invalid', false), undefined);
});
