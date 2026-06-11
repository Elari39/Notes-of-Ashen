export interface User {
  id: number;
  account: string;
  email: string;
  avatarUrl: string;
  nickname: string;
  role: string;
  status: string;
  createdAt: string;
  updatedAt: string;
}

export interface TokenPair {
  accessToken: string;
  refreshToken: string;
  tokenType: string;
  expiresIn: number;
}

export interface SiteSettings {
  registrationEnabled: boolean;
  homeArticleLayout: HomeArticleLayout;
  siteTitle: string;
  siteDescription: string;
  siteKeywords: string;
  siteBaseUrl: string;
  resumePageEnabled: boolean;
  resumeNavHidden: boolean;
  projectsPageEnabled: boolean;
  projectsNavHidden: boolean;
}

export type HomeArticleLayout = 'standard' | 'alternating';

export interface ResumePage {
  title: string;
  subtitle: string;
  contentMarkdown: string;
  experiences: ResumeExperience[];
  educations: ResumeEducation[];
  skills: ResumeSkill[];
}

export interface ResumeExperience {
  id?: number;
  role: string;
  organization: string;
  location: string;
  startDate: string;
  endDate: string;
  description: string;
  highlights: string[];
  displayOrder: number;
}

export interface ResumeEducation {
  id?: number;
  school: string;
  degree: string;
  major: string;
  location: string;
  startDate: string;
  endDate: string;
  description: string;
  highlights: string[];
  displayOrder: number;
}

export interface ResumeSkill {
  id?: number;
  category: string;
  name: string;
  level: number;
  description: string;
  displayOrder: number;
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
}

export interface Tag {
  id: number;
  name: string;
  slug: string;
  description: string;
  createdBy: number;
  createdAt: string;
  updatedAt: string;
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
  status: string;
  viewCount: number;
  likeCount: number;
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
  status: string;
  viewCount: number;
  likeCount: number;
  scheduledAt?: string;
  publishedAt?: string;
  isPinned: boolean;
  displayPriority: number;
  seoTitle: string;
  seoDescription: string;
  seoKeywords: string;
  tagIds: number[];
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

export interface GeoStat {
  countryCode: string;
  countryName: string;
  regionName: string;
  cityName: string;
  pv: number;
  uv: number;
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
  geoStats: GeoStat[];
  popularArticles: Article[];
  recentArticles: Article[];
  recentLogs: Log[];
}

export interface Log {
  id: number;
  userId?: number;
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
