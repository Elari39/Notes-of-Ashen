import type { AxiosError } from 'axios';

type ErrorResponse = {
  code?: number;
  message?: string;
};

type Language = 'zh' | 'en';
type LocalizedText = Record<Language, string>;

// 前端自构造的错误标识符（稳定 key，避免中文短语拼串作 key 的脆弱性）。
// 后端返回的英文 message 仍作为 exactMessages 的 key 直接匹配。
export const ERROR_KEYS = {
  sessionExpired: '__session_expired__',
  operationFailed: '__operation_failed__',
  timeout: '__timeout__',
  timeoutWrite: '__timeout_write__',
  network: '__network_error__',
  duplicateSubmit: '__duplicate_submit__',
} as const;

export class AppError extends Error {
  code?: number;
  status?: number;

  constructor(message: string, code?: number, status?: number) {
    super(message);
    this.name = 'AppError';
    this.code = code;
    this.status = status;
  }
}

const localized = (zh: string, en: string): LocalizedText => ({ zh, en });

const fieldNames: Record<string, LocalizedText> = {
  account: localized('账号', 'Account'),
  password: localized('密码', 'Password'),
  email: localized('邮箱', 'Email'),
  nickname: localized('昵称', 'Nickname'),
  avatarUrl: localized('头像 URL', 'Avatar URL'),
  oldPassword: localized('旧密码', 'Old password'),
  newPassword: localized('新密码', 'New password'),
  refreshToken: localized('登录凭证', 'Refresh token'),
  title: localized('标题', 'Title'),
  slug: localized('路径', 'Slug'),
  content: localized('正文', 'Content'),
  coverUrl: localized('封面 URL', 'Cover URL'),
  baseUrl: localized('Base URL', 'Base URL'),
  model: localized('模型', 'Model'),
  apiKey: localized('API Key', 'API key'),
  firstByteTimeoutSeconds: localized('首字等待时间', 'First byte timeout'),
  streamTimeoutSeconds: localized('流式输出超时', 'Streaming timeout'),
  nonStreamTimeoutSeconds: localized('非流式输出超时', 'Non-streaming timeout'),
  status: localized('状态', 'Status'),
  q: localized('搜索关键词', 'Search keyword'),
  repoUrl: localized('仓库 URL', 'Repo URL'),
  demoUrl: localized('演示 URL', 'Demo URL'),
  referrer: localized('来源', 'Referrer'),
  versionNo: localized('版本号', 'Version number'),
  code: localized('验证码', 'Code'),
  role: localized('角色', 'Role'),
  action: localized('操作', 'Action'),
  displayPriority: localized('显示优先级', 'Display priority'),
  homeArticleLayout: localized('首页布局', 'Home article layout'),
  path: localized('路径', 'Path'),
  routeType: localized('路由类型', 'Route type'),
  markdownFile: localized('Markdown 文件', 'Markdown file'),
  items: localized('项目', 'Items'),
  experiences: localized('工作经历', 'Experiences'),
  educations: localized('教育经历', 'Educations'),
  skills: localized('技能', 'Skills'),
  highlights: localized('亮点', 'Highlights'),
  tags: localized('标签', 'Tags'),
};

const exactMessages: Record<string, LocalizedText> = {
  'avatarUrl format is invalid': localized('头像 URL 必须以 http:// 或 https:// 开头', 'Avatar URL must start with http:// or https://'),
  'coverUrl format is invalid': localized('封面 URL 必须以 http:// 或 https:// 开头，或留空', 'Cover URL must start with http:// or https://, or be empty'),
  'email format is invalid': localized('邮箱格式不正确', 'Email format is invalid'),
  'account or email already exists': localized('账号或邮箱已存在', 'Account or email already exists'),
  'email already exists': localized('邮箱已存在', 'Email already exists'),
  'account or password is incorrect': localized('账号或密码不正确', 'Account or password is incorrect'),
  'user is disabled': localized('账号已被禁用', 'This account has been disabled'),
  'old password is incorrect': localized('旧密码不正确', 'Old password is incorrect'),
  'registration is disabled': localized('注册已关闭，请联系管理员。', 'Registration is disabled. Please contact the administrator.'),
  'refresh token is invalid': localized('登录状态无效，请重新登录', 'Your session is invalid. Please sign in again.'),
  'refresh token is expired': localized('登录已过期，请重新登录', 'Your session has expired. Please sign in again.'),
  'captcha is required': localized('请填写图形验证码', 'Please enter the captcha'),
  'captcha is expired': localized('图形验证码已过期，请重新获取', 'The captcha has expired. Please refresh it.'),
  'captcha is incorrect': localized('图形验证码不正确', 'The captcha is incorrect'),
  'email code is required': localized('请填写邮箱验证码', 'Please enter the email verification code'),
  'email code is expired': localized('邮箱验证码已过期，请重新获取', 'The email verification code has expired. Please request a new one.'),
  'email code is incorrect': localized('邮箱验证码不正确', 'The email verification code is incorrect'),
  'verify code was sent recently': localized('验证码发送过于频繁，请稍后再试', 'A verification code was sent recently. Please try again later.'),
  'purpose is invalid': localized('验证用途不正确，请刷新后重试', 'The verification purpose is invalid. Please refresh and try again.'),
  'too many requests': localized('请求过于频繁，请稍后再试', 'Too many requests. Please try again later.'),
  'missing authorization header': localized('请先登录', 'Please sign in first'),
  'invalid authorization header': localized('登录凭证格式不正确', 'The authorization header is invalid'),
  'invalid or expired token': localized('登录已过期，请重新登录', 'Your session has expired. Please sign in again.'),
  'resource not found': localized('资源不存在或已被删除', 'The resource does not exist or has been removed'),
  'resource already exists': localized('资源已存在', 'The resource already exists'),
  'article slug already exists': localized('文章路径已存在', 'The article slug already exists'),
  'category already exists': localized('分类名称或路径已存在', 'The category name or slug already exists'),
  'tag already exists': localized('标签名称或路径已存在', 'The tag name or slug already exists'),
  'tag not found': localized('标签不存在', 'Tag not found'),
  'ai assistant is disabled': localized('AI 辅助未启用，请先在后台 AI 配置中启用。', 'AI assistance is disabled. Enable it in AI Settings first.'),
  'ai assistant is not configured': localized('AI 辅助尚未配置，请检查 Base URL、API Key 和模型。', 'AI assistance is not configured. Check Base URL, API key, and model.'),
  'ai response is invalid': localized('AI 返回内容格式不正确，请重试。', 'The AI response format is invalid. Please try again.'),
  'baseUrl format is invalid': localized('Base URL 必须以 http:// 或 https:// 开头，或留空', 'Base URL must start with http:// or https://, or be empty'),
  'firstByteTimeoutSeconds is invalid': localized('首字等待时间必须在 1 到 1800 秒之间', 'First byte timeout must be between 1 and 1800 seconds'),
  'streamTimeoutSeconds is invalid': localized('流式输出超时必须在 1 到 1800 秒之间', 'Streaming timeout must be between 1 and 1800 seconds'),
  'nonStreamTimeoutSeconds is invalid': localized('非流式输出超时必须在 1 到 1800 秒之间', 'Non-streaming timeout must be between 1 and 1800 seconds'),
  'streamTimeoutSeconds must be greater than or equal to firstByteTimeoutSeconds': localized('流式输出超时不能小于首字等待时间', 'Streaming timeout cannot be lower than first byte timeout'),
  'nonStreamTimeoutSeconds must be greater than or equal to firstByteTimeoutSeconds': localized('非流式输出超时不能小于首字等待时间', 'Non-streaming timeout cannot be lower than first byte timeout'),
  'cannot manage other user\'s article': localized('不能管理其他用户的文章', 'You cannot manage another user\'s article'),
  'content manager permission required': localized('需要编辑或管理员权限', 'Editor or administrator permission is required'),
  'admin permission required': localized('需要管理员权限', 'Administrator permission is required'),
  'cannot disable yourself': localized('不能禁用自己的账号', 'You cannot disable your own account'),
  'cannot downgrade yourself': localized('不能降低自己的管理员权限', 'You cannot downgrade your own administrator role'),
  'at least one active admin is required': localized('至少需要保留一个可用管理员', 'At least one active administrator is required'),
  'email is unchanged': localized('新邮箱不能与当前邮箱相同', 'The new email cannot be the same as the current email'),
  'internal server error': localized('服务暂时不可用，请稍后重试', 'The service is temporarily unavailable. Please try again later.'),
  'invalid id': localized('ID 不合法', 'The ID is invalid'),
  'invalid request body or parameters': localized('请求参数不合法', 'The request body or parameters are invalid'),
  'invalid user': localized('用户信息无效，请重新登录', 'The user info is invalid. Please sign in again.'),
  'missing user': localized('用户信息缺失，请重新登录', 'User info is missing. Please sign in again.'),
  'article not found': localized('文章不存在或已被删除', 'The article does not exist or has been removed'),
  'refresh token is required': localized('登录状态异常，请重新登录', 'Your session is invalid. Please sign in again.'),
  'markdown file is invalid': localized('Markdown 文件不合法', 'The Markdown file is invalid'),
  'markdown file is required': localized('请选择 Markdown 文件', 'A Markdown file is required'),
  'markdown file must be .md': localized('文件必须是 .md 后缀', 'The file must have a .md extension'),
  'markdown file is too large': localized('Markdown 文件过大', 'The Markdown file is too large'),
  'markdown content is empty after title extraction': localized('正文为空（标题提取后无内容）', 'The Markdown content is empty after title extraction'),
  'markdown content is empty': localized('正文内容为空', 'The Markdown content is empty'),
  'front matter is invalid': localized('Front Matter 格式不正确', 'The front matter is invalid'),
  'feature disabled': localized('该功能未开启', 'This feature is disabled'),
  'email service is disabled': localized('邮件服务未开启', 'The email service is disabled'),
  'email service is not configured': localized('邮件服务未配置', 'The email service is not configured'),
  'rate limiter unavailable': localized('限流服务暂时不可用，请稍后重试', 'The rate limiter is unavailable. Please try again later.'),
  'too many likes for this article': localized('点赞过于频繁，请稍后再试', 'Too many likes for this article. Please try again later.'),
  'items id is duplicated': localized('项目 ID 不能重复', 'Item IDs must not be duplicated'),
  'ai key encryption secret is not configured': localized('AI 密钥加密配置缺失，请联系管理员', 'The AI key encryption secret is not configured. Please contact the administrator.'),
  'role is invalid': localized('角色不合法', 'The role is invalid'),
  'action is invalid': localized('操作类型不合法', 'The action is invalid'),
  'displayPriority is invalid': localized('显示优先级不合法', 'The display priority is invalid'),
  'homeArticleLayout is invalid': localized('首页布局不合法', 'The home article layout is invalid'),
  'versionNo is invalid': localized('版本号不合法', 'The version number is invalid'),
  'referrer length is invalid': localized('来源长度不符合要求', 'The referrer length is invalid'),
  'code length is invalid': localized('验证码长度不符合要求', 'The code length is invalid'),
  'success': localized('操作成功', 'Success'),
  [ERROR_KEYS.sessionExpired]: localized('登录已过期，请重新登录', 'Your session has expired. Please sign in again.'),
  [ERROR_KEYS.operationFailed]: localized('操作失败，请稍后重试', 'Operation failed. Please try again later.'),
  [ERROR_KEYS.timeout]: localized('请求超时，请稍后重试', 'The request timed out. Please try again later.'),
  [ERROR_KEYS.timeoutWrite]: localized(
    '网络较慢，操作可能仍在处理中，请稍后刷新页面确认，避免重复提交',
    'The network is slow. Your request may still be processing — please refresh the page later to verify, to avoid duplicate submissions.',
  ),
  [ERROR_KEYS.network]: localized('网络连接异常，请检查后重试', 'Network connection failed. Please check your connection and try again.'),
  [ERROR_KEYS.duplicateSubmit]: localized('请勿重复提交，正在处理中…', 'Please do not submit again — your request is still being processed.'),
};

const defaultFallbacks: Record<Language, string> = {
  zh: '操作失败，请稍后重试',
  en: 'Operation failed. Please try again later.',
};

const readLanguage = (): Language => {
  if (typeof localStorage !== 'undefined' && localStorage.getItem('notesOfAshen.language') === 'en') {
    return 'en';
  }
  if (typeof document !== 'undefined' && document.documentElement.lang.toLowerCase().startsWith('en')) {
    return 'en';
  }
  return 'zh';
};

const textFor = (value: LocalizedText | undefined, language: Language, fallback: string) => value?.[language] || fallback;

const translateMessage = (message?: string, fallback?: string) => {
  const language = readLanguage();
  const fallbackText = (fallback && exactMessages[fallback]?.[language]) || fallback || defaultFallbacks[language];
  const raw = message?.trim();
  if (!raw) return fallbackText;
  if (exactMessages[raw]) return textFor(exactMessages[raw], language, fallbackText);

  const requiredMatch = raw.match(/^(.+) is required$/);
  if (requiredMatch) {
    const field = textFor(fieldNames[requiredMatch[1]], language, requiredMatch[1]);
    return language === 'zh' ? `${field}不能为空` : `${field} is required`;
  }

  const lengthMatch = raw.match(/^(.+) length is invalid$/);
  if (lengthMatch) {
    const field = textFor(fieldNames[lengthMatch[1]], language, lengthMatch[1]);
    return language === 'zh' ? `${field}长度不符合要求` : `${field} length is invalid`;
  }

  // 必须在 "is invalid" 之前匹配，避免被泛化吃掉
  const localAddressMatch = raw.match(/^(.+) must not point to a local address$/);
  if (localAddressMatch) {
    const field = textFor(fieldNames[localAddressMatch[1]], language, localAddressMatch[1]);
    return language === 'zh' ? `${field}不能指向本地地址` : `${field} must not point to a local address`;
  }

  // 必须在 "is invalid" 之前匹配，产出更通顺的字段数量提示
  const countMatch = raw.match(/^(.+) count is invalid$/);
  if (countMatch) {
    const field = textFor(fieldNames[countMatch[1]], language, countMatch[1]);
    return language === 'zh' ? `${field}数量不合法` : `${field} count is invalid`;
  }

  const invalidMatch = raw.match(/^(.+) is invalid$/);
  if (invalidMatch) {
    const field = textFor(fieldNames[invalidMatch[1]], language, invalidMatch[1]);
    return language === 'zh' ? `${field}不合法` : `${field} is invalid`;
  }

  // 安全网：未命中任何翻译时，中文页面回退到已本地化的 fallbackText（调用方 i18n 文案或默认兜底），
  // 绝不把英文原文展示在中文页面；英文页面保留原文。
  return language === 'zh' ? fallbackText : raw;
};

// 鸭子类型判断类 axios 错误，避免运行时硬耦合 axios 实例（降低与 HTTP 客户端的耦合）。
type HttpErrorLike = {
  code?: string;
  config?: { method?: string };
  response?: { status?: number; data?: ErrorResponse };
};

const isHttpErrorLike = (error: unknown): error is HttpErrorLike => {
  if (typeof error !== 'object' || error === null) {
    return false;
  }
  const candidate = error as Record<string, unknown>;
  // axios 错误对象特征：携带 config 或 response 字段，且本身是 Error 实例。
  return error instanceof Error && ('config' in candidate || 'response' in candidate);
};

export const toAppError = (error: unknown, fallback?: string) => {
  if (error instanceof AppError) {
    return new AppError(translateMessage(error.message, fallback), error.code, error.status);
  }

  if (isHttpErrorLike(error)) {
    const httpError = error as AxiosError<ErrorResponse> & HttpErrorLike;
    const method = (httpError.config?.method || 'get').toLowerCase();
    const isWrite = method !== 'get';
    if (httpError.code === 'ECONNABORTED') {
      // 写操作超时大概率服务端仍在处理；提示用户稍后刷新核对，而不是反复提交
      const key = isWrite ? ERROR_KEYS.timeoutWrite : ERROR_KEYS.timeout;
      return new AppError(translateMessage(key, fallback), undefined, httpError.response?.status);
    }
    if (!httpError.response) {
      return new AppError(translateMessage(ERROR_KEYS.network, fallback));
    }

    const data = httpError.response.data;
    const message = translateMessage(data?.message, fallback);
    return new AppError(message, data?.code, httpError.response.status);
  }

  if (error instanceof Error) {
    return new AppError(translateMessage(error.message, fallback));
  }

  return new AppError(fallback || '操作失败，请稍后重试');
};

export const getErrorMessage = (error: unknown, fallback?: string) => toAppError(error, fallback).message;
