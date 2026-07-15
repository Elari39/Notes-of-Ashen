import assert from 'node:assert/strict';
import test from 'node:test';

import {
  buildArticleCompletionPatch,
  readAITaxonomySuggestions,
  type ArticleCompletionFields,
} from '../src/pages/admin/articleAICompletionPolicy.ts';

const emptyFields: ArticleCompletionFields = {
  title: '',
  slug: '',
  summary: '',
  seoTitle: '',
  seoDescription: '',
  seoKeywords: '',
};

test('文章 AI 补全会填充全部空字段', () => {
  const result = buildArticleCompletionPatch(emptyFields, {
    title: ' 标题 ',
    slug: 'article-title',
    summary: '摘要',
    seoTitle: 'SEO 标题',
    seoDescription: 'SEO 描述',
    seoKeywords: 'Go,AI',
  });
  assert.deepEqual(result, {
    patch: {
      title: '标题',
      slug: 'article-title',
      summary: '摘要',
      seoTitle: 'SEO 标题',
      seoDescription: 'SEO 描述',
      seoKeywords: 'Go,AI',
    },
    appliedCount: 6,
  });
});

test('文章 AI 补全不会覆盖已有字段或写入空结果', () => {
  const result = buildArticleCompletionPatch(
    { ...emptyFields, title: '人工标题', seoDescription: '人工描述' },
    { title: 'AI 标题', slug: '  ', seoDescription: 'AI 描述', summary: 'AI 摘要' },
  );
  assert.deepEqual(result, { patch: { summary: 'AI 摘要' }, appliedCount: 1 });
});

test('分类标签仅形成去空去重的文字建议', () => {
  assert.deepEqual(readAITaxonomySuggestions({
    categorySuggestion: ' 技术 ',
    tagSuggestions: [' Go ', '', 'Go', 'AI', '后端'],
  }), {
    category: '技术',
    tags: ['Go', 'AI', '后端'],
  });
  assert.equal(readAITaxonomySuggestions({}), null);
});
