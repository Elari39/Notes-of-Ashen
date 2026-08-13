export type UserRole = 'user' | 'editor' | 'admin';

export type UserStatus = 'active' | 'disabled';

export type ArticleStatus = 'draft' | 'published' | 'archived';

export interface User {
  id: number;
  account: string;
  email: string;
  avatarUrl: string;
  nickname: string;
  role: UserRole;
  status: UserStatus;
  createdAt: string;
  updatedAt: string;
}

export interface TokenPair {
  accessToken: string;
  // refreshToken 由后端 HttpOnly Cookie 下发，响应体中通常省略此字段。
  refreshToken?: string;
  tokenType: string;
  expiresIn: number;
}

export interface SiteSettings {
  registrationEnabled: boolean;
  registrationEmailCodeRequired: boolean;
  homeArticleLayout: HomeArticleLayout;
  homeCtaHidden: boolean;
  siteTitle: string;
  siteDescription: string;
  siteKeywords: string;
  siteBaseUrl: string;
  projectsPageEnabled: boolean;
  projectsNavHidden: boolean;
  /** RAG 问答页是否对外开放；未返回时前端按关闭处理。 */
  ragChatPageEnabled?: boolean;
  /** RAG 问答页是否在前台导航中显示。 */
  ragChatNavHidden?: boolean;
  /** 访问 /ask 所需的最低角色。 */
  ragChatAccessLevel?: RAGChatAccessLevel;
}

export type HomeArticleLayout = 'standard' | 'alternating';

export type RAGChatAccessLevel = 'guest' | 'user' | 'editor';

export type RAGChatMessageRole = 'user' | 'assistant';

export interface RAGChatSource {
  articleId: number;
  title: string;
  /** 后端可返回绝对 URL；缺失时前端回退到 /article/:articleId。 */
  url?: string;
  snippet?: string;
}

export interface RAGChatMessage {
  id: string;
  sessionId?: string;
  role: RAGChatMessageRole;
  content: string;
  sources?: RAGChatSource[];
  /** 引用文章仍公开但正文已更新；旧回答保留，旧片段不再展示。 */
  sourcesUpdated?: boolean;
  /** 客户端主动停止或流式上游意外中断时的局部回答。 */
  incomplete?: boolean;
  hiddenAt?: string;
  createdAt: string;
}

export interface RAGChatSession {
  id: string;
  title: string;
  sourceEpoch?: number;
  expiresAt?: string;
  createdAt: string;
  updatedAt: string;
  messages?: RAGChatMessage[];
}

export interface ProjectItem {
  id: string;
  tagIds?: number[];
  title: string;
  summary: string;
  role: string;
  period: string;
  tags: string[];
  coverUrl: string;
  demoUrl: string;
  repoUrl: string;
  contentMarkdown: string;
  featured: boolean;
}

export interface ProjectsPage {
  title: string;
  subtitle: string;
  items: ProjectItem[];
}

export interface Category {
  id: number;
  name: string;
  slug: string;
  description: string;
  createdBy: number;
  createdAt: string;
  updatedAt: string;
  articleCount: number;
}

export interface Tag {
  id: number;
  name: string;
  slug: string;
  description: string;
  createdBy: number;
  createdAt: string;
  updatedAt: string;
  articleCount: number;
}

export interface Article {
  id: number;
  authorId: number;
  categoryId?: number;
  title: string;
  slug: string;
  summary: string;
  content?: string;
  coverUrl: string;
  status: ArticleStatus;
  viewCount: number;
  likeCount: number;
  wordCount: number;
  readingTimeMinutes: number;
  scheduledAt?: string;
  publishedAt?: string;
  isPinned: boolean;
  displayPriority: number;
  seoTitle: string;
  seoDescription: string;
  seoKeywords: string;
  createdAt: string;
  updatedAt: string;
  tags?: Tag[];
  category?: Category;
  searchHighlights?: ArticleSearchHighlights;
}

export interface ArticleSearchHighlights {
  title?: string;
  summary?: string;
  content?: string;
}

export interface ArticleContext {
  previous?: Article;
  next?: Article;
  related: Article[];
}

export interface ArticleVersion {
  id: number;
  articleId: number;
  versionNo: number;
  changedBy: number;
  authorId: number;
  categoryId?: number;
  title: string;
  slug: string;
  summary: string;
  content?: string;
  coverUrl: string;
  status: ArticleStatus;
  viewCount: number;
  likeCount: number;
  scheduledAt?: string;
  publishedAt?: string;
  isPinned: boolean;
  displayPriority: number;
  seoTitle: string;
  seoDescription: string;
  seoKeywords: string;
  tagIds?: number[];
  originalCreatedAt?: string;
  originalUpdatedAt?: string;
  createdAt: string;
}

export interface TrafficTrendPoint {
  date: string;
  pv: number;
  uv: number;
}

export interface RefererStat {
  sourceType: string;
  sourceName: string;
  pv: number;
}

export interface AdminStats {
  articleTotal: number;
  publishedTotal: number;
  draftTotal: number;
  archivedTotal: number;
  scheduledTotal: number;
  viewTotal: number;
  likeTotal: number;
  todayPv: number;
  todayUv: number;
  userTotal: number;
  categoryTotal: number;
  tagTotal: number;
  trafficTrend: TrafficTrendPoint[];
  topReferers: RefererStat[];
  popularArticles: Article[];
  recentArticles: Article[];
  recentLogs: Log[];
}

export type SearchSuggestionKind = 'article' | 'category' | 'tag';

export interface SearchSuggestion {
  kind: SearchSuggestionKind;
  id: number;
  label: string;
  articleCount?: number;
}

export interface MediaAsset {
  id: number;
  storageKey: string;
  url: string;
  originalName: string;
  mimeType: string;
  sizeBytes: number;
  width: number;
  height: number;
  altText: string;
  sha256: string;
  createdBy: number;
  createdAt: string;
  updatedAt: string;
}

export interface AnalyticsSummary {
  pv: number; uv: number; likes: number;
  previousPv: number; previousUv: number; previousLikes: number;
  pvChange?: number; uvChange?: number; likesChange?: number;
}

export interface PageAnalytics {
  routeType: string; path: string; articleId?: number; title?: string; pv: number; uv: number;
}

export interface AnalyticsOverview {
  from: string; to: string; summary: AnalyticsSummary;
  trend: TrafficTrendPoint[]; topPages: PageAnalytics[]; topReferers: RefererStat[];
}

export interface ArticleAnalytics {
  articleId: number; title: string; status: ArticleStatus;
  pv: number; uv: number; likes: number; totalViews: number; totalLikes: number;
}

export interface ArticleAnalyticsPoint extends TrafficTrendPoint { likes: number; }

export interface ArticleAnalyticsDetail {
  article: ArticleAnalytics; from: string; to: string;
  trend: ArticleAnalyticsPoint[]; referers: RefererStat[];
}

export type DependencyStatus = 'up' | 'down' | 'disabled';
export interface DependencyCheck { name: string; status: DependencyStatus; latencyMs: number; message?: string; }
export interface SystemHealth { status: 'healthy' | 'degraded'; checkedAt: string; checks: DependencyCheck[]; }

export interface BackupRestoreResult { users: number; articles: number; media: number; warnings: string[]; }

export interface Log {
  id: number;
  userId?: number;
  userAccount?: string;
  eventType: string;
  resourceType: string;
  resourceId?: number;
  metadata?: string;
  ip: string;
  userAgent: string;
  createdAt: string;
}

export interface PaginatedResp<T> {
  items: T[];
  total: number;
  page: number;
  size: number;
}

export interface BaseResp<T = unknown> {
  code: number;
  message: string;
  data: T;
}

export interface NoDataResp {
  code: number;
  message: string;
  data?: never;
}
