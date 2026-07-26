import assert from 'node:assert/strict';
import { test } from 'node:test';
import { containsMath } from '../src/utils/markdownMath.ts';

test('空内容不触发数学公式加载', () => {
  assert.equal(containsMath(''), false);
});

test('普通 Markdown 不触发数学公式加载', () => {
  assert.equal(containsMath('# 标题\n\n正文段落，包含 `code` 和 [链接](https://example.com)。'), false);
});

test('行内 $ 公式触发加载', () => {
  assert.equal(containsMath('质能方程 $E = mc^2$ 很有名。'), true);
});

test('块级 $$ 公式触发加载', () => {
  assert.equal(containsMath('$$\n\\int_0^1 x^2 dx\n$$'), true);
});

test('LaTeX 括号语法触发加载', () => {
  assert.equal(containsMath('公式 \\(a+b\\) 与 \\[c+d\\]。'), true);
});

test('美元价格文本按宽松策略触发加载（只多加载资源，不影响渲染）', () => {
  assert.equal(containsMath('价格是 $5。'), true);
});

test('普通反斜杠转义不触发加载', () => {
  assert.equal(containsMath('路径 C:\\Users\\test 和转义 \\*star\\*。'), false);
});
