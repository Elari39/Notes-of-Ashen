import axios, { AxiosError } from 'axios';

type ErrorResponse = {
  code?: number;
  message?: string;
};

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

const fieldNames: Record<string, string> = {
  account: '账号',
  password: '密码',
  email: '邮箱',
  nickname: '昵称',
  avatarUrl: '头像 URL',
  oldPassword: '旧密码',
  newPassword: '新密码',
  refreshToken: '登录凭证',
  title: '标题',
  slug: '路径',
  content: '正文',
  coverUrl: '封面 URL',
  status: '状态',
  q: '搜索关键词',
};

const exactMessages: Record<string, string> = {
  'avatarUrl format is invalid': '头像 URL 必须以 http:// 或 https:// 开头',
  'coverUrl format is invalid': '封面 URL 必须以 http:// 或 https:// 开头，或留空',
  'email format is invalid': '邮箱格式不正确',
  'account or email already exists': '账号或邮箱已存在',
  'email already exists': '邮箱已存在',
  'account or password is incorrect': '账号或密码不正确',
  'user is disabled': '账号已被禁用',
  'old password is incorrect': '旧密码不正确',
  'registration is disabled': '注册已关闭，请联系管理员。',
  'refresh token is invalid': '登录状态无效，请重新登录',
  'refresh token is expired': '登录已过期，请重新登录',
  'captcha is required': '请填写图形验证码',
  'captcha is expired': '图形验证码已过期，请重新获取',
  'captcha is incorrect': '图形验证码不正确',
  'email code is required': '请填写邮箱验证码',
  'email code is expired': '邮箱验证码已过期，请重新获取',
  'email code is incorrect': '邮箱验证码不正确',
  'verify code was sent recently': '验证码发送过于频繁，请稍后再试',
  'purpose is invalid': '验证用途不正确，请刷新后重试',
  'too many requests': '请求过于频繁，请稍后再试',
  'missing authorization header': '请先登录',
  'invalid authorization header': '登录凭证格式不正确',
  'invalid or expired token': '登录已过期，请重新登录',
  'resource not found': '资源不存在或已被删除',
  'resource already exists': '资源已存在',
  'article slug already exists': '文章路径已存在',
  'category already exists': '分类名称或路径已存在',
  'tag already exists': '标签名称或路径已存在',
  'tag not found': '标签不存在',
  'cannot manage other user\'s article': '不能管理其他用户的文章',
  'content manager permission required': '需要编辑或管理员权限',
  'admin permission required': '需要管理员权限',
  'cannot disable yourself': '不能禁用自己的账号',
  'cannot downgrade yourself': '不能降低自己的管理员权限',
  'at least one active admin is required': '至少需要保留一个可用管理员',
  'email is unchanged': '新邮箱不能与当前邮箱相同',
  'internal server error': '服务暂时不可用，请稍后重试',
};

const translateMessage = (message?: string, fallback = '操作失败，请稍后重试') => {
  const raw = message?.trim();
  if (!raw) return fallback;
  if (exactMessages[raw]) return exactMessages[raw];

  const requiredMatch = raw.match(/^(.+) is required$/);
  if (requiredMatch) {
    return `${fieldNames[requiredMatch[1]] || requiredMatch[1]}不能为空`;
  }

  const lengthMatch = raw.match(/^(.+) length is invalid$/);
  if (lengthMatch) {
    return `${fieldNames[lengthMatch[1]] || lengthMatch[1]}长度不符合要求`;
  }

  const invalidMatch = raw.match(/^(.+) is invalid$/);
  if (invalidMatch) {
    return `${fieldNames[invalidMatch[1]] || invalidMatch[1]}不合法`;
  }

  return raw;
};

export const toAppError = (error: unknown, fallback?: string) => {
  if (error instanceof AppError) {
    return error;
  }

  if (axios.isAxiosError(error)) {
    const axiosError = error as AxiosError<ErrorResponse>;
    if (axiosError.code === 'ECONNABORTED') {
      return new AppError('请求超时，请稍后重试', undefined, axiosError.response?.status);
    }
    if (!axiosError.response) {
      return new AppError('网络连接异常，请检查后重试');
    }

    const data = axiosError.response.data;
    const message = translateMessage(data?.message, fallback);
    return new AppError(message, data?.code, axiosError.response.status);
  }

  if (error instanceof Error) {
    return new AppError(translateMessage(error.message, fallback));
  }

  return new AppError(fallback || '操作失败，请稍后重试');
};

export const getErrorMessage = (error: unknown, fallback?: string) => toAppError(error, fallback).message;
