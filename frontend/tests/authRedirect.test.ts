import assert from 'node:assert/strict';
import test from 'node:test';
import { resolveAuthRedirect } from '../src/utils/authRedirect.ts';

test('会话失效的字符串与受保护路由的 Location 都保留路径、查询和片段', () => {
  assert.equal(resolveAuthRedirect('/admin/articles?page=2#draft'), '/admin/articles?page=2#draft');
  assert.equal(resolveAuthRedirect({ pathname: '/admin/articles', search: '?page=2', hash: '#draft', key: 'k' }), '/admin/articles?page=2#draft');
  assert.equal(resolveAuthRedirect({ pathname: '/profile' }), '/profile');
});

test('拒绝外站、相对路径、控制字符、畸形类型和登录循环', () => {
  for (const target of [undefined, null, {}, 42, '/login', '/login/?next=/admin', '/admin/../login',
    'https://outside.invalid', '//outside.invalid', '/\\outside.invalid', '/\noutside.invalid',
    'admin/articles', { pathname: '/admin', search: () => '?' }, { pathname: '/admin', hash: 'evil' }]) {
    assert.equal(resolveAuthRedirect(target), '/', JSON.stringify(target));
  }
});
