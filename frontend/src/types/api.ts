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
  emailCode: string;
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
  refreshToken: string;
}

export type CaptchaPurpose = 'login' | 'register' | 'reset_password' | 'change_password' | 'update_email';

export type VerifyCodePurpose = Exclude<CaptchaPurpose, 'login'>;

export interface CaptchaReq {
  purpose: CaptchaPurpose;
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
  registrationEnabled: boolean;
  homeArticleLayout: 'standard' | 'alternating';
  siteTitle: string;
  siteDescription: string;
  siteKeywords: string;
  siteBaseUrl: string;
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
  status: string; // 'active' | 'disabled'
}

export interface UpdateUserRoleReq {
  role: string; // 'user' | 'editor' | 'admin'
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
  scheduledAt?: string;
  seoTitle?: string;
  seoDescription?: string;
  seoKeywords?: string;
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
