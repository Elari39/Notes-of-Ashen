import http from '../utils/http';
import type { AxiosResponse } from 'axios';
import { BaseResp, Article, ArticleContext, ArticleVersion, ArticleStatus, PaginatedResp } from '../types';
import {
  AIAssistReq,
  AIAssistResp,
  ArticleListParams,
  CreateArticleReq,
  UpdateArticleReq,
  UpdateArticleStatusReq,
  ArticleLikeResp,
  SearchReindexResp,
} from '../types/api';

export const getArticles = (params?: ArticleListParams, signal?: AbortSignal) =>
  http.get<unknown, BaseResp<PaginatedResp<Article>>>('/articles', { params, signal });

export const getAdminArticles = (params?: ArticleListParams, signal?: AbortSignal) =>
  http.get<unknown, BaseResp<PaginatedResp<Article>>>('/admin/articles', { params, signal });

export const getArticleById = (id: number | string, signal?: AbortSignal) =>
  http.get<unknown, BaseResp<Article>>(`/articles/${id}`, { signal });

export const getArticlePreview = (id: number | string, signal?: AbortSignal) =>
  http.get<unknown, BaseResp<Article>>(`/articles/${id}/preview`, { signal });

export const getArticleContext = (id: number | string, signal?: AbortSignal) =>
  http.get<unknown, BaseResp<ArticleContext>>(`/articles/${id}/context`, { signal });

export const likeArticle = (id: number | string) =>
  http.post<unknown, BaseResp<ArticleLikeResp>>(`/articles/${id}/like`);

export const createArticle = (data: CreateArticleReq) => 
  http.post<unknown, BaseResp<Article>>('/articles', data);

export const updateArticle = (id: number | string, data: UpdateArticleReq) => 
  http.put<unknown, BaseResp<Article>>(`/articles/${id}`, data);

export const deleteArticle = (id: number | string) => 
  http.delete<unknown, BaseResp>(`/articles/${id}`);

export const updateArticleStatus = (id: number | string, status: ArticleStatus) =>
  http.patch<unknown, BaseResp<Article>>(`/articles/${id}/status`, { status } as UpdateArticleStatusReq);

export const assistArticle = (data: AIAssistReq) =>
  http.post<unknown, BaseResp<AIAssistResp>>('/articles/ai/assist', data, { timeout: 600000 });

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

export const getArticleVersions = (id: number | string, params?: { page?: number; size?: number }, signal?: AbortSignal) =>
  http.get<unknown, BaseResp<PaginatedResp<ArticleVersion>>>(`/articles/${id}/versions`, { params, signal });

export const getArticleVersion = (id: number | string, versionNo: number | string, signal?: AbortSignal) =>
  http.get<unknown, BaseResp<ArticleVersion>>(`/articles/${id}/versions/${versionNo}`, { signal });

export const restoreArticleVersion = (id: number | string, versionNo: number | string) =>
  http.post<unknown, BaseResp<Article>>(`/articles/${id}/versions/${versionNo}/restore`);

export const reindexArticleSearch = () =>
  http.post<unknown, BaseResp<SearchReindexResp>>('/admin/search/reindex');

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
