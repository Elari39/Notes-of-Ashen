import http from '../utils/http';
import type { AxiosResponse } from 'axios';
import { BaseResp, Article, ArticleContext, ArticleVersion, PaginatedResp } from '../types';
import {
  AIAssistReq,
  AIAssistResp,
  ArticleListParams,
  CreateArticleReq,
  UpdateArticleReq,
  UpdateArticleStatusReq,
} from '../types/api';

export const getArticles = (params?: ArticleListParams) => 
  http.get<unknown, BaseResp<PaginatedResp<Article>>>('/articles', { params });

export const getAdminArticles = (params?: ArticleListParams) => 
  http.get<unknown, BaseResp<PaginatedResp<Article>>>('/admin/articles', { params });

export const getArticleById = (id: number | string) => 
  http.get<unknown, BaseResp<Article>>(`/articles/${id}`);

export const getArticlePreview = (id: number | string) =>
  http.get<unknown, BaseResp<Article>>(`/articles/${id}/preview`);

export const getArticleContext = (id: number | string) =>
  http.get<unknown, BaseResp<ArticleContext>>(`/articles/${id}/context`);

export const createArticle = (data: CreateArticleReq) => 
  http.post<unknown, BaseResp<Article>>('/articles', data);

export const updateArticle = (id: number | string, data: UpdateArticleReq) => 
  http.put<unknown, BaseResp<Article>>(`/articles/${id}`, data);

export const deleteArticle = (id: number | string) => 
  http.delete<unknown, BaseResp>(`/articles/${id}`);

export const updateArticleStatus = (id: number | string, status: string) => 
  http.patch<unknown, BaseResp<Article>>(`/articles/${id}/status`, { status } as UpdateArticleStatusReq);

export const assistArticle = (data: AIAssistReq) =>
  http.post<unknown, BaseResp<AIAssistResp>>('/articles/ai/assist', data);

export const importMarkdownArticle = (file: File) => {
  const form = new FormData();
  form.append('file', file);
  return http.post<unknown, BaseResp<Article>>('/articles/import', form);
};

export const exportArticleMarkdown = async (id: number | string) => {
  const response = await http.get<unknown, AxiosResponse<Blob>>(`/articles/${id}/export`, {
    responseType: 'blob',
  });
  return {
    blob: response.data,
    filename: filenameFromDisposition(response.headers['content-disposition']) || `article-${id}.md`,
  };
};

export const getArticleVersions = (id: number | string, params?: { page?: number; size?: number }) =>
  http.get<unknown, BaseResp<PaginatedResp<ArticleVersion>>>(`/articles/${id}/versions`, { params });

export const getArticleVersion = (id: number | string, versionNo: number | string) =>
  http.get<unknown, BaseResp<ArticleVersion>>(`/articles/${id}/versions/${versionNo}`);

export const restoreArticleVersion = (id: number | string, versionNo: number | string) =>
  http.post<unknown, BaseResp<Article>>(`/articles/${id}/versions/${versionNo}/restore`);

const filenameFromDisposition = (value?: string) => {
  if (!value) {
    return '';
  }
  const utf8Match = value.match(/filename\*=UTF-8''([^;]+)/i);
  if (utf8Match) {
    return decodeURIComponent(utf8Match[1]);
  }
  const match = value.match(/filename="?([^";]+)"?/i);
  return match?.[1] || '';
};
