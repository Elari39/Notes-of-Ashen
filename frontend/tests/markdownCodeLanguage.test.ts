import assert from 'node:assert/strict';
import test from 'node:test';

import { resolveMarkdownCodeLanguage } from '../src/utils/markdownCodeLanguage.ts';

test('识别已注册语言并保留未知语言的纯文本降级能力', () => {
  assert.equal(resolveMarkdownCodeLanguage('language-java'), 'java');
  assert.equal(resolveMarkdownCodeLanguage('language-toml'), 'toml');
  assert.equal(resolveMarkdownCodeLanguage('language-mermaid'), 'mermaid');
  assert.equal(resolveMarkdownCodeLanguage('language-unknown'), 'unknown');
  assert.equal(resolveMarkdownCodeLanguage('inline-code'), '');
  assert.equal(resolveMarkdownCodeLanguage(), '');
});

test('兼容现有语言别名', () => {
  assert.equal(resolveMarkdownCodeLanguage('language-golang'), 'go');
  assert.equal(resolveMarkdownCodeLanguage('language-js'), 'javascript');
  assert.equal(resolveMarkdownCodeLanguage('language-ts'), 'typescript');
  assert.equal(resolveMarkdownCodeLanguage('language-py'), 'python');
  assert.equal(resolveMarkdownCodeLanguage('language-shell'), 'bash');
  assert.equal(resolveMarkdownCodeLanguage('language-yml'), 'yaml');
  assert.equal(resolveMarkdownCodeLanguage('language-mmd'), 'mermaid');
});

test('完整解析包含特殊字符和常见文件名的语言别名', () => {
  assert.equal(resolveMarkdownCodeLanguage('code language-C++ highlighted'), 'cpp');
  assert.equal(resolveMarkdownCodeLanguage('language-c#'), 'csharp');
  assert.equal(resolveMarkdownCodeLanguage('language-cs'), 'csharp');
  assert.equal(resolveMarkdownCodeLanguage('language-dockerfile'), 'docker');
  assert.equal(resolveMarkdownCodeLanguage('language-kt'), 'kotlin');
  assert.equal(resolveMarkdownCodeLanguage('language-rs'), 'rust');
  assert.equal(resolveMarkdownCodeLanguage('language-html'), 'markup');
  assert.equal(resolveMarkdownCodeLanguage('language-xml'), 'markup');
  assert.equal(resolveMarkdownCodeLanguage('language-pwsh'), 'powershell');
});
