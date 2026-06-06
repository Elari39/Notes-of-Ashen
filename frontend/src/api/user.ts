import http from '../utils/http';
import { BaseResp, User, Log, PaginatedResp } from '../types';
import { UpdateUserReq, UpdatePasswordReq, UpdateUserStatusReq, UpdateUserRoleReq, PageParams } from '../types/api';

export const getCurrentUser = () => 
  http.get<unknown, BaseResp<User>>('/users/me');

export const updateCurrentUser = (data: UpdateUserReq) => 
  http.put<unknown, BaseResp<User>>('/users/me', data);

export const updatePassword = (data: UpdatePasswordReq) => 
  http.put<unknown, BaseResp>('/users/me/password', data);

// Admin
export const getUsers = (params?: PageParams) => 
  http.get<unknown, BaseResp<PaginatedResp<User>>>('/admin/users', { params });

export const updateUserStatus = (id: number, status: string) => 
  http.patch<unknown, BaseResp>(`/admin/users/${id}/status`, { status } as UpdateUserStatusReq);

export const updateUserRole = (id: number, role: string) =>
  http.patch<unknown, BaseResp>(`/admin/users/${id}/role`, { role } as UpdateUserRoleReq);

export const getLogs = (params?: PageParams) => 
  http.get<unknown, BaseResp<PaginatedResp<Log>>>('/admin/logs', { params });
