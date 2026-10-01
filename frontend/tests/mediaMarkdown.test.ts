import assert from 'node:assert/strict';
import test from 'node:test';
import { insertMediaMarkdown } from '../src/utils/mediaMarkdown.ts';

const asset = { url: '/media/image.png', altText: 'Image', originalName: 'image.png' };

test('在关闭代码围栏后插图不会破坏图表代码', () => {
  const code = '```mermaid\nflowchart LR\nA --> B\n```';
  const inserted = insertMediaMarkdown(code, code.length, code.length, asset);
  assert.equal(inserted.content, `${code}\n\n![Image](/media/image.png)`);
  assert.equal(inserted.cursor, inserted.content.length);
});

test('插图保留前后正文、选择替换与现有段落边界', () => {
  assert.equal(insertMediaMarkdown('leftRIGHT', 4, 4, asset).content, 'left\n\n![Image](/media/image.png)\n\nRIGHT');
  assert.equal(insertMediaMarkdown('replace', 0, 7, asset).content, '![Image](/media/image.png)');
  assert.equal(insertMediaMarkdown('left\n\nright', 6, 6, asset).content, 'left\n\n![Image](/media/image.png)\n\nright');
});

test('媒体说明中的方括号与换行作为文字处理', () => {
  const inserted = insertMediaMarkdown('', 0, 0, { ...asset, altText: 'one]\n[two' });
  assert.equal(inserted.content, '![one\\] \\[two](/media/image.png)');
});
