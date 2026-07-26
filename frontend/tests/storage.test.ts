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

test('按前缀清理同时移除内存镜像和底层存储键', () => {
  const values = new Map<string, string>();
  const storage = createSafeStorage(() => ({
    getItem: (key) => values.get(key) ?? null,
    setItem: (key, value) => { values.set(key, value); },
    removeItem: (key) => { values.delete(key); },
    get length() { return values.size; },
    key: (index: number) => Array.from(values.keys())[index] ?? null,
  }));

  storage.setItem('draft:user:1:new', 'a');
  storage.setItem('draft:user:1:42', 'b');
  storage.setItem('draft:user:2:new', 'c');
  storage.removeByPrefix('draft:user:1:');

  assert.equal(storage.getItem('draft:user:1:new'), null);
  assert.equal(storage.getItem('draft:user:1:42'), null);
  assert.equal(storage.getItem('draft:user:2:new'), 'c');
  assert.deepEqual(Array.from(values.keys()), ['draft:user:2:new']);
});
