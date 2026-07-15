import assert from 'node:assert/strict';
import test from 'node:test';

import { buildArchiveTree, type ArchiveArticle } from '../src/utils/archiveTree.ts';

const article = (
  id: number,
  title: string,
  publishedAt: string | undefined,
  createdAt: string,
): ArchiveArticle => ({ id, title, publishedAt, createdAt });

test('按年月日倒序分组，并在同日内按时间和 id 倒序排列', () => {
  const tree = buildArchiveTree([
    article(1, 'old', '2025-12-31T12:00:00', '2025-12-01T12:00:00'),
    article(2, 'morning', '2026-07-15T08:00:00', '2026-07-01T12:00:00'),
    article(4, 'same time higher id', '2026-07-15T20:00:00', '2026-07-01T12:00:00'),
    article(3, 'same time lower id', '2026-07-15T20:00:00', '2026-07-01T12:00:00'),
    article(5, 'previous month', '2026-06-30T12:00:00', '2026-06-01T12:00:00'),
  ]);

  assert.deepEqual(tree.years.map((node) => node.year), [2026, 2025]);
  assert.deepEqual(tree.years[0].months.map((node) => node.month), [7, 6]);
  assert.deepEqual(tree.years[0].months[0].days[0].articles.map((item) => item.id), [4, 3, 2]);
  assert.equal(tree.articleCount, 5);
});

test('发布时间无效时回退创建时间，完全无效时归入日期未知', () => {
  const tree = buildArchiveTree([
    article(1, 'fallback', 'invalid', '2026-01-02T12:00:00'),
    article(2, 'undated', undefined, 'invalid'),
  ]);

  assert.equal(tree.years[0].year, 2026);
  assert.equal(tree.years[0].months[0].days[0].articles[0].id, 1);
  assert.deepEqual(tree.undated.map((item) => item.id), [2]);
});

test('按文章 id 去重，并默认展开最新年月日分支', () => {
  const tree = buildArchiveTree([
    article(7, 'duplicate first', '2026-07-15T12:00:00', '2026-07-01T12:00:00'),
    article(7, 'duplicate last', '2026-07-16T12:00:00', '2026-07-01T12:00:00'),
    article(8, 'older', '2025-01-01T12:00:00', '2025-01-01T12:00:00'),
  ]);

  assert.equal(tree.articleCount, 2);
  assert.equal(tree.years[0].months[0].days[0].articles[0].title, 'duplicate last');
  assert.deepEqual(tree.defaultExpandedKeys, ['year:2026', 'month:2026-07', 'day:2026-07-16']);
});

test('只有无日期文章时默认展开日期未知分组', () => {
  const tree = buildArchiveTree([article(1, 'undated', undefined, 'invalid')]);
  assert.deepEqual(tree.defaultExpandedKeys, ['undated']);
});
