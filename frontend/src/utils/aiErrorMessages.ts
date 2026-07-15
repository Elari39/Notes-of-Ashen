export type AIErrorLanguage = 'zh' | 'en';

type AIErrorMessage = Record<AIErrorLanguage, string>;

const localized = (zh: string, en: string): AIErrorMessage => ({ zh, en });

export const AI_EXACT_ERROR_MESSAGES: Record<string, AIErrorMessage> = {
  'unsupported ai api format': localized('暂不支持该 AI 接口格式', 'This AI API format is not supported'),
  'ai model is required': localized('请先填写要测试的模型', 'Enter a model before testing'),
  'ai response body exceeds limit': localized('AI 服务响应内容过大', 'The AI provider response is too large'),
  'ai probe response is not json': localized('AI 服务探测响应不是有效的 JSON', 'The AI provider probe response is not valid JSON'),
  'ai response has no choices': localized('AI 服务响应中没有可用结果', 'The AI provider response contains no choices'),
  'ai response has no text content': localized('AI 服务响应中没有文本内容', 'The AI provider response contains no text content'),
  'ai provider request failed': localized('AI 服务请求失败，请检查接口格式、地址、密钥和模型', 'The AI provider request failed. Check the API format, URL, key, and model.'),
  'ai provider request timed out': localized('AI 服务请求超时，请检查超时设置后重试', 'The AI provider request timed out. Check the timeout settings and try again.'),
  'ai api key needs update': localized('已保存的 API Key 需要重新录入', 'The saved API key must be re-entered'),
  'api key is required': localized('请填写 API Key', 'API key is required'),
  'api key is required for unsaved ai endpoint': localized('当前接口格式或地址尚未保存，请填写对应的 API Key', 'Enter an API key for the unsaved API format or URL'),
  'api key must be replaced or cleared when ai endpoint changes': localized('修改接口格式或地址时，请同时录入新 API Key 或清空旧密钥', 'When changing the API format or URL, replace or clear the saved API key'),
  'apiKey and clearApiKey cannot be used together': localized('不能同时录入并清空 API Key', 'An API key cannot be entered and cleared at the same time'),
  'baseUrl is required when AI is enabled': localized('启用 AI 时必须填写 Base URL', 'Base URL is required when AI is enabled'),
  'model is required when AI is enabled': localized('启用 AI 时必须填写模型', 'A model is required when AI is enabled'),
  'apiKey is required when AI is enabled': localized('启用 AI 时必须配置 API Key', 'An API key is required when AI is enabled'),
  'auth access secret is not configured': localized('AI 密钥加密配置缺失，请联系管理员', 'The AI key encryption configuration is missing. Please contact the administrator.'),
};

export const getAIExactErrorMessage = (
  message: string,
  language: AIErrorLanguage,
): string | undefined => AI_EXACT_ERROR_MESSAGES[message]?.[language];
