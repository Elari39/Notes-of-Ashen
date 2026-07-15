import assert from 'node:assert/strict';
import test from 'node:test';
import { collectPaginated } from '../src/utils/pagination.ts';

test('collectPaginated 收集超过一页的全部条目', async () => {
  const requestedPages: number[] = [];
  const items = await collectPaginated(async ({ page, size }) => {
    requestedPages.push(page);
    const start = (page - 1) * size;
    const pageItems = Array.from({ length: Math.min(size, 205 - start) }, (_, index) => start + index + 1);
    return { data: { items: pageItems, total: 205, page, size } };
  }, { pageSize: 100 });

  assert.equal(items.length, 205);
  assert.deepEqual(requestedPages, [1, 2, 3]);
  assert.equal(items.at(-1), 205);
});

test('collectPaginated 在翻页之间响应中止信号', async () => {
  const controller = new AbortController();
  let calls = 0;
  await assert.rejects(
    collectPaginated(async ({ page, size }) => {
      calls += 1;
      controller.abort();
      return { data: { items: [1], total: 2, page, size } };
    }, { signal: controller.signal }),
    { name: 'AbortError' },
  );
  assert.equal(calls, 1);
});
