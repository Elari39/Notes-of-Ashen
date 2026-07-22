import http from '../utils/http';
import { BaseResp, NoDataResp, Category, PaginatedResp } from '../types';
import { CreateCategoryReq, UpdateCategoryReq, PageParams } from '../types/api';

export const getCategories = (params?: PageParams, signal?: AbortSignal) =>
  http.get<unknown, BaseResp<PaginatedResp<Category>>>('/categories', { params, signal });

export const getAdminCategories = (params?: PageParams, signal?: AbortSignal) =>
  http.get<unknown, BaseResp<PaginatedResp<Category>>>('/admin/categories', { params, signal });

export const createCategory = (data: CreateCategoryReq) => 
  http.post<unknown, BaseResp<Category>>('/categories', data);

export const updateCategory = (id: number | string, data: UpdateCategoryReq) => 
  http.put<unknown, BaseResp<Category>>(`/categories/${id}`, data);

export const deleteCategory = (id: number | string) => 
  http.delete<unknown, NoDataResp>(`/categories/${id}`);
