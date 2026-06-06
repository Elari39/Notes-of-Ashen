import http from '../utils/http';
import { BaseResp, Category, PaginatedResp } from '../types';
import { CreateCategoryReq, UpdateCategoryReq, PageParams } from '../types/api';

export const getCategories = (params?: PageParams) => 
  http.get<any, BaseResp<PaginatedResp<Category>>>('/categories', { params });

export const createCategory = (data: CreateCategoryReq) => 
  http.post<any, BaseResp<Category>>('/categories', data);

export const updateCategory = (id: number | string, data: UpdateCategoryReq) => 
  http.put<any, BaseResp<Category>>(`/categories/${id}`, data);

export const deleteCategory = (id: number | string) => 
  http.delete<any, BaseResp>(`/categories/${id}`);
