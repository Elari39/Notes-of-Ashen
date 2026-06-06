import http from '../utils/http';
import { BaseResp, Article, PaginatedResp } from '../types';
import { CreateArticleReq, UpdateArticleReq, UpdateArticleStatusReq, PageParams } from '../types/api';

export const getArticles = (params?: PageParams) => 
  http.get<any, BaseResp<PaginatedResp<Article>>>('/articles', { params });

export const getArticleById = (id: number | string) => 
  http.get<any, BaseResp<Article>>(`/articles/${id}`);

export const createArticle = (data: CreateArticleReq) => 
  http.post<any, BaseResp<Article>>('/articles', data);

export const updateArticle = (id: number | string, data: UpdateArticleReq) => 
  http.put<any, BaseResp<Article>>(`/articles/${id}`, data);

export const deleteArticle = (id: number | string) => 
  http.delete<any, BaseResp>(`/articles/${id}`);

export const updateArticleStatus = (id: number | string, status: string) => 
  http.patch<any, BaseResp<Article>>(`/articles/${id}/status`, { status } as UpdateArticleStatusReq);
