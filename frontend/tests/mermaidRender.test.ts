import assert from 'node:assert/strict';
import test from 'node:test';
import { hasMermaidRenderError } from '../src/utils/mermaidRender.ts';

test('识别 Mermaid 返回的错误 SVG', () => {
  assert.equal(
    hasMermaidRenderError('<svg><text class="error-text">Syntax error in text</text></svg>'),
    true,
  );
  assert.equal(
    hasMermaidRenderError('<svg><text class="nodeLabel">正常图表</text></svg>'),
    false,
  );
});

test('普通 SVG 文本不会被误判为渲染错误', () => {
  assert.equal(hasMermaidRenderError('<svg><text>Syntax documentation</text></svg>'), false);
});
