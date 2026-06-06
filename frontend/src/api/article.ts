import http from '../utils/http';
import { BaseResp, Article, ArticleContext, ArticleVersion, PaginatedResp } from '../types';
import { ArticleListParams, CreateArticleReq, UpdateArticleReq, UpdateArticleStatusReq } from '../types/api';

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

export const getArticleVersions = (id: number | string, params?: { page?: number; size?: number }) =>
  http.get<unknown, BaseResp<PaginatedResp<ArticleVersion>>>(`/articles/${id}/versions`, { params });

export const getArticleVersion = (id: number | string, versionNo: number | string) =>
  http.get<unknown, BaseResp<ArticleVersion>>(`/articles/${id}/versions/${versionNo}`);

export const restoreArticleVersion = (id: number | string, versionNo: number | string) =>
  http.post<unknown, BaseResp<Article>>(`/articles/${id}/versions/${versionNo}/restore`);
