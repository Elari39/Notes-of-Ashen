import http from '../utils/http';
import { BaseResp, Tag, PaginatedResp } from '../types';
import { CreateTagReq, UpdateTagReq, PageParams } from '../types/api';

export const getTags = (params?: PageParams) => 
  http.get<any, BaseResp<PaginatedResp<Tag>>>('/tags', { params });

export const createTag = (data: CreateTagReq) => 
  http.post<any, BaseResp<Tag>>('/tags', data);

export const updateTag = (id: number | string, data: UpdateTagReq) => 
  http.put<any, BaseResp<Tag>>(`/tags/${id}`, data);

export const deleteTag = (id: number | string) => 
  http.delete<any, BaseResp>(`/tags/${id}`);
