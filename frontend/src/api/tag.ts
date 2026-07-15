import http from '../utils/http';
import { BaseResp, Tag, PaginatedResp } from '../types';
import { CreateTagReq, UpdateTagReq, PageParams } from '../types/api';

export const getTags = (params?: PageParams, signal?: AbortSignal) =>
  http.get<unknown, BaseResp<PaginatedResp<Tag>>>('/tags', { params, signal });

export const createTag = (data: CreateTagReq) => 
  http.post<unknown, BaseResp<Tag>>('/tags', data);

export const updateTag = (id: number | string, data: UpdateTagReq) => 
  http.put<unknown, BaseResp<Tag>>(`/tags/${id}`, data);

export const deleteTag = (id: number | string) => 
  http.delete<unknown, BaseResp>(`/tags/${id}`);
