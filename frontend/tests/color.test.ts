import assert from 'node:assert/strict';
import test from 'node:test';
import { contrastRatio, getContrastingTextColor, hexToRgb, hexToRgba, isHexColor } from '../src/utils/color.ts';

test('识别并解析六位十六进制颜色', () => {
  assert.equal(isHexColor('#cc785c'), true);
  assert.equal(isHexColor('cc785c'), true);
  assert.equal(isHexColor('#fff'), false);
  assert.deepEqual(hexToRgb('#cc785c'), [204, 120, 92]);
  assert.equal(hexToRgba('#cc785c', 0.15), 'rgba(204, 120, 92, 0.15)');
});

test('默认珊瑚色选择暖黑前景并满足正文对比要求', () => {
  const foreground = getContrastingTextColor('#cc785c');
  assert.equal(foreground, '#141413');
  assert.ok(contrastRatio(foreground, '#cc785c') >= 4.5);
});

test('深色强调色选择浅色前景', () => {
  assert.equal(getContrastingTextColor('#2b2b2b'), '#faf9f5');
});
