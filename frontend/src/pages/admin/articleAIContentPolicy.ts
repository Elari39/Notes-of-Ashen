import type { AIAssistAction } from '../../types/api.ts';
import { MAX_AI_FULL_ARTICLE_CONTENT_BYTES, utf8ByteLength } from '../../utils/utf8.ts';

const fullArticleAIActions = new Set<AIAssistAction>(['complete', 'metadata']);

/** 与服务端全文 AI 输入上限保持一致；改写类操作仍由其 30,000 字符限制处理。 */
export const exceedsFullArticleAIContentLimit = (action: AIAssistAction, content: string) => (
  fullArticleAIActions.has(action)
  && utf8ByteLength(content.trim()) > MAX_AI_FULL_ARTICLE_CONTENT_BYTES
);
