import assert from 'node:assert/strict';
import test from 'node:test';
import { createSafeStorage } from '../src/utils/storage.ts';

test('存储 API 抛出 SecurityError 时降级为内存状态', () => {
  const storage = createSafeStorage(() => {
    throw new DOMException('blocked', 'SecurityError');
  });

  storage.setItem('language', 'en');
  assert.equal(storage.getItem('language'), 'en');
  storage.removeItem('language');
  assert.equal(storage.getItem('language'), null);
});

test('底层写入失败后仍可读取内存值', () => {
  const storage = createSafeStorage(() => ({
    getItem: () => null,
    setItem: () => { throw new Error('quota exceeded'); },
    removeItem: () => undefined,
  }));

  storage.setItem('liked', '1');
  assert.equal(storage.getItem('liked'), '1');
});
