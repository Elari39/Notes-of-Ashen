/**
 * API Request Payloads
 */

export interface PageParams {
  page?: number;
  size?: number;
  [key: string]: string | number | boolean | undefined;
}

// Auth
export interface RegisterReq {
  account: string;
  password: string;
  email: string;
  nickname?: string;
  avatarUrl?: string;
}

export interface LoginReq {
  account: string;
  password: string;
}

export interface RefreshReq {
  refreshToken: string;
}

export interface UpdateSiteSettingsReq {
  registrationEnabled: boolean;
  homeArticleLayout: 'standard' | 'alternating';
}

// User
export interface UpdateUserReq {
  email?: string;
  avatarUrl?: string;
  nickname?: string;
}

export interface UpdatePasswordReq {
  oldPassword: string;
  newPassword: string;
}

// Admin
export interface UpdateUserStatusReq {
  status: string; // 'active' | 'disabled'
}

export interface ArticleListParams extends PageParams {
  q?: string;
  categoryId?: number;
  tagId?: number;
  status?: string;
}

// Article
export interface CreateArticleReq {
  categoryId?: number;
  title: string;
  slug: string;
  summary?: string;
  content: string;
  coverUrl?: string;
  status?: string; // 'draft' | 'published' | 'archived'
  tagIds?: number[];
}

export interface UpdateArticleReq extends Partial<CreateArticleReq> {}

export interface UpdateArticleStatusReq {
  status: string;
}

// Category
export interface CreateCategoryReq {
  name: string;
  slug: string;
  description?: string;
}

export interface UpdateCategoryReq extends Partial<CreateCategoryReq> {}

// Tag
export interface CreateTagReq {
  name: string;
  slug: string;
  description?: string;
}

export interface UpdateTagReq extends Partial<CreateTagReq> {}
