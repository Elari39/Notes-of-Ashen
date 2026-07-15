import assert from 'node:assert/strict';
import test from 'node:test';
import { routeUsesOwnSEO } from '../src/utils/routeSeo.ts';

test('内容路由拥有独立 SEO，不由 Layout 默认值覆盖', () => {
  for (const pathname of ['/archive', '/archive/', '/search', '/search/', '/projects', '/projects/', '/article/1', '/admin/preview/1']) {
    assert.equal(routeUsesOwnSEO(pathname), true, pathname);
  }
  assert.equal(routeUsesOwnSEO('/'), false);
  assert.equal(routeUsesOwnSEO('/login'), false);
  assert.equal(routeUsesOwnSEO('/projects-disabled'), false);
});
