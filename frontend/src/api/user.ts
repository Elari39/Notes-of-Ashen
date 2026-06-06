import http from '../utils/http';
import { BaseResp, User, Log, PaginatedResp } from '../types';
import { UpdateUserReq, UpdatePasswordReq, UpdateUserStatusReq, PageParams } from '../types/api';

export const getCurrentUser = () => 
  http.get<any, BaseResp<User>>('/users/me');

export const updateCurrentUser = (data: UpdateUserReq) => 
  http.put<any, BaseResp<User>>('/users/me', data);

export const updatePassword = (data: UpdatePasswordReq) => 
  http.put<any, BaseResp>('/users/me/password', data);

// Admin
export const getUsers = (params?: PageParams) => 
  http.get<any, BaseResp<PaginatedResp<User>>>('/admin/users', { params });

export const updateUserStatus = (id: number, status: string) => 
  http.patch<any, BaseResp>(`/admin/users/${id}/status`, { status } as UpdateUserStatusReq);

export const getLogs = (params?: PageParams) => 
  http.get<any, BaseResp<PaginatedResp<Log>>>('/admin/logs', { params });
