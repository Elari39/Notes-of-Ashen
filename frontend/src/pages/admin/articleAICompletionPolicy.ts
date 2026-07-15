import type { AIAssistResp } from '../../types/api';

export interface ArticleCompletionFields {
  title: string;
  slug: string;
  summary: string;
  seoTitle: string;
  seoDescription: string;
  seoKeywords: string;
}

export interface AITaxonomySuggestions {
  category: string;
  tags: string[];
}

const completionFields: Array<keyof ArticleCompletionFields> = [
  'title',
  'slug',
  'summary',
  'seoTitle',
  'seoDescription',
  'seoKeywords',
];

export const buildArticleCompletionPatch = (
  current: ArticleCompletionFields,
  response: AIAssistResp,
): { patch: Partial<ArticleCompletionFields>; appliedCount: number } => {
  const patch: Partial<ArticleCompletionFields> = {};
  for (const field of completionFields) {
    const generated = response[field]?.trim();
    if (!current[field].trim() && generated) {
      patch[field] = generated;
    }
  }
  return { patch, appliedCount: Object.keys(patch).length };
};

export const readAITaxonomySuggestions = (response: AIAssistResp): AITaxonomySuggestions | null => {
  const category = response.categorySuggestion?.trim() || '';
  const tags = [...new Set((response.tagSuggestions || []).map((tag) => tag.trim()).filter(Boolean))].slice(0, 5);
  return category || tags.length > 0 ? { category, tags } : null;
};
