import { test } from 'node:test';
import assert from 'node:assert/strict';
import { selectHomeArticles } from '../src/utils/homeArticles.ts';

test('首页推荐去重保持服务端顺序且不修改输入', () => {
  const articles = [{ id: 9 }, { id: 3 }, { id: 5 }, { id: 9 }];
  const result = selectHomeArticles(articles, true);
  assert.equal(result.featuredArticle, articles[0]);
  assert.deepEqual(result.listArticles.map(({ id }) => id), [3, 5]);
  assert.equal(articles.length, 4);
});
test('筛选及后续页保留完整列表，单篇与空列表无虚假条目', () => {
  const articles = [{ id: 9 }];
  assert.deepEqual(selectHomeArticles(articles, false), { featuredArticle: undefined, listArticles: articles });
  assert.deepEqual(selectHomeArticles(articles, true).listArticles, []);
  assert.deepEqual(selectHomeArticles([], true), { featuredArticle: undefined, listArticles: [] });
});
