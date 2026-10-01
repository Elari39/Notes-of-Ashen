import { test } from 'node:test';
import assert from 'node:assert/strict';
import { resolveThemeStyle, themeStyles, themeStyleAccents } from '../src/store/themeStyles.ts';

test('六套主题保持兼容，未知偏好回退原有刊物', () => {
  assert.equal(themeStyles.length, 6);
  for (const style of themeStyles) {
    assert.equal(resolveThemeStyle(style), style);
    assert.match(themeStyleAccents[style].light, /^#[0-9a-f]{6}$/);
    assert.match(themeStyleAccents[style].dark, /^#[0-9a-f]{6}$/);
  }
  assert.equal(resolveThemeStyle(null), 'editorial');
  assert.equal(resolveThemeStyle('unknown'), 'editorial');
});
