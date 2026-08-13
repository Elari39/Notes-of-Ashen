/**
 * API Request Payloads
 */

import type { ArticleStatus, RAGChatAccessLevel, UserRole, UserStatus } from './index';

export interface PageParams {
  page?: number;
  size?: number;
  [key: string]: string | number | boolean | undefined;
}

export interface OperationLogListParams extends PageParams {
  eventType?: string;
  actor?: string;
  ip?: string;
  startAt?: string;
  endAt?: string;
}

// Auth
export interface RegisterReq {
  account: string;
  password: string;
  email: string;
  emailCode?: string;
  nickname?: string;
  avatarUrl?: string;
}

export interface LoginReq {
  account: string;
  password: string;
  captchaId: string;
  captchaCode: string;
}

export interface RefreshReq {
  // refreshToken 现由后端 HttpOnly Cookie 携带；此字段可选，仅兼容直接调用 API 的客户端。
  refreshToken?: string;
}

export type CaptchaPurpose = 'login' | 'register' | 'reset_password' | 'change_password' | 'update_email';

export type VerifyCodePurpose = Exclude<CaptchaPurpose, 'login'>;

export interface CaptchaReq {
  purpose?: CaptchaPurpose;
}

export interface CaptchaResp {
  captchaId: string;
  imageData: string;
  expiresIn: number;
}

export interface SendVerifyCodeReq {
  email: string;
  purpose: 'register' | 'reset_password';
  captchaId: string;
  captchaCode: string;
}

export interface ResetPasswordReq {
  email: string;
  emailCode: string;
  newPassword: string;
}

export interface UpdateSiteSettingsReq {
  registrationEnabled?: boolean;
  homeArticleLayout?: 'standard' | 'alternating';
  homeCtaHidden?: boolean;
  siteTitle?: string;
  siteDescription?: string;
  siteKeywords?: string;
  siteBaseUrl?: string;
  projectsPageEnabled?: boolean;
  projectsNavHidden?: boolean;
  ragChatPageEnabled?: boolean;
  ragChatNavHidden?: boolean;
  ragChatAccessLevel?: RAGChatAccessLevel;
}

export interface RAGSettingsResp {
  enabled: boolean;
  chatBaseUrl: string;
  embeddingBaseUrl: string;
  rerankUrl: string;
  apiKeyConfigured: boolean;
  apiKeyNeedsUpdate: boolean;
  chatModel: string;
  embeddingModel: string;
  embeddingDimensions: number;
  rerankModel: string;
  historyRetentionDays: number;
}

/** RAG 私有会话只支持这些固定保留期，0 表示永久保留。 */
export const RAG_HISTORY_RETENTION_OPTIONS = [0, 7, 30, 60, 90, 180, 365] as const;

export const isValidRAGHistoryRetentionDays = (value: number): boolean => (
  Number.isInteger(value) && RAG_HISTORY_RETENTION_OPTIONS.includes(value as typeof RAG_HISTORY_RETENTION_OPTIONS[number])
);

export interface UpdateRAGSettingsReq {
  enabled: boolean;
  chatBaseUrl?: string;
  embeddingBaseUrl?: string;
  rerankUrl?: string;
  apiKey?: string;
  clearApiKey?: boolean;
  chatModel?: string;
  embeddingModel?: string;
  embeddingDimensions?: number;
  rerankModel?: string;
  historyRetentionDays?: number;
}

export type RAGTestKind = 'chat' | 'embedding' | 'rerank';

/** 测试接口只使用已保存的加密配置，避免浏览器回传或记录 API Key。 */
export interface RAGTestReq {
  kind: RAGTestKind;
}

export interface RAGTestResp {
  latencyMs: number;
  message?: string;
  embeddingDimensions?: number;
}

export type RAGIndexStatus = 'needs_rebuild' | 'rebuilding' | 'ready' | 'error' | 'disabled';

export interface RAGStatusResp {
  status: RAGIndexStatus;
  enabled: boolean;
  configured: boolean;
  queueDepth?: number;
  indexedArticles?: number;
  indexedChunks?: number;
  lastError?: string;
}

export interface RAGChatStreamReq {
  question: string;
  sessionId?: string;
}

export interface ArticleLikeResp {
  liked: boolean;
  likeCount: number;
}

export interface SearchReindexResp {
  indexed: number;
  enabled: boolean;
}

export interface ProjectItemReq {
  id: string;
  tagIds?: number[];
  title: string;
  summary: string;
  role: string;
  period: string;
  coverUrl: string;
  demoUrl: string;
  repoUrl: string;
  contentMarkdown: string;
  featured: boolean;
}

export interface UpdateProjectsPageReq {
  title: string;
  subtitle?: string;
  items: ProjectItemReq[];
}

// User
export interface UpdateUserReq {
  email?: string;
  emailCode?: string;
  avatarUrl?: string;
  nickname?: string;
}

export interface UpdatePasswordReq {
  oldPassword: string;
  newPassword: string;
  emailCode: string;
}

export interface UserVerifyCodeReq {
  email?: string;
  purpose: 'change_password' | 'update_email';
  captchaId: string;
  captchaCode: string;
}

// Admin
export interface UpdateUserStatusReq {
  status: UserStatus;
}

export interface UpdateUserRoleReq {
  role: UserRole;
}

export interface AnalyticsParams extends PageParams {
  from?: string;
  to?: string;
  q?: string;
}

export interface BackupExportReq {
  currentPassword: string;
  passphrase: string;
}

export interface ArticleListParams extends PageParams {
  q?: string;
  categoryId?: number;
  tagId?: number;
  // 'scheduled' 为前端派生筛选值（后端识别为 published 且 scheduled_at 在未来）。
  status?: ArticleStatus | 'scheduled';
}

// Article
export interface CreateArticleReq {
  categoryId?: number;
  title: string;
  slug: string;
  summary?: string;
  content: string;
  coverUrl?: string;
  status?: ArticleStatus;
  scheduledAt?: string;
  isPinned?: boolean;
  displayPriority?: number;
  seoTitle?: string;
  seoDescription?: string;
  seoKeywords?: string;
  tagIds?: number[];
}

// 更新接口的 title、slug、content 为后端实际必填字段，其余字段按接口定义可选。
export interface UpdateArticleReq {
  categoryId?: number;
  title: string;
  slug: string;
  summary?: string;
  content: string;
  coverUrl?: string;
  status?: ArticleStatus;
  scheduledAt?: string;
  isPinned?: boolean;
  displayPriority?: number;
  seoTitle?: string;
  seoDescription?: string;
  seoKeywords?: string;
  tagIds?: number[];
}

export interface UpdateArticleStatusReq {
  status: ArticleStatus;
}

export type AIAssistAction = 'complete' | 'metadata' | 'proofread' | 'polish' | 'expand' | 'shorten' | 'translate';

export interface AIAssistReq {
  action: AIAssistAction;
  title?: string;
  content: string;
}

export interface AIAssistResp {
  title?: string;
  slug?: string;
  summary?: string;
  seoTitle?: string;
  seoDescription?: string;
  seoKeywords?: string;
  categorySuggestion?: string;
  tagSuggestions?: string[];
  revisedContent?: string;
  suggestions?: string[];
}

export type AIAPIFormat = 'openai' | 'anthropic';

export interface AISettingsResp {
  enabled: boolean;
  apiFormat: AIAPIFormat;
  baseUrl: string;
  model: string;
  apiKeyConfigured: boolean;
  apiKeyNeedsUpdate: boolean;
  firstByteTimeoutSeconds: number;
  nonStreamTimeoutSeconds: number;
}

export interface UpdateAISettingsReq {
  enabled: boolean;
  apiFormat?: AIAPIFormat;
  baseUrl?: string;
  apiKey?: string;
  clearApiKey?: boolean;
  model?: string;
  firstByteTimeoutSeconds?: number;
  nonStreamTimeoutSeconds?: number;
}

export interface AIConnectionReq {
  apiFormat: AIAPIFormat;
  baseUrl: string;
  apiKey?: string;
  firstByteTimeoutSeconds: number;
  nonStreamTimeoutSeconds: number;
}

export interface AIModelsResp {
  models: string[];
}

export interface AIModelTestReq extends AIConnectionReq {
  model: string;
}

export interface AIModelTestResp {
  model: string;
  latencyMs: number;
}

export interface TrafficVisitReq {
  path: string;
  routeType: string;
  articleId?: number;
  referrer?: string;
}

// Category
export interface CreateCategoryReq {
  name: string;
  slug: string;
  description?: string;
}

export interface UpdateCategoryReq {
  name: string;
  slug: string;
  description?: string;
}

// Tag
export interface CreateTagReq {
  name: string;
  slug: string;
  description?: string;
}

export interface UpdateTagReq {
  name: string;
  slug: string;
  description?: string;
}
