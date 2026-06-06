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
}

export type HomeArticleLayout = 'standard' | 'alternating';

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
  publishedAt?: string;
  createdAt: string;
  updatedAt: string;
  tags?: Tag[];
  category?: Category;
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
