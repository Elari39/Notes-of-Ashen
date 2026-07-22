import http from '../utils/http';
import { BaseResp, NoDataResp, User, Log, UserRole, UserStatus, PaginatedResp } from '../types';
import {
  PageParams,
  OperationLogListParams,
  UpdatePasswordReq,
  UpdateUserReq,
  UpdateUserRoleReq,
  UpdateUserStatusReq,
  UserVerifyCodeReq,
} from '../types/api';

export const getCurrentUser = () => 
  http.get<unknown, BaseResp<User>>('/users/me');

export const updateCurrentUser = (data: UpdateUserReq) => 
  http.put<unknown, BaseResp<User>>('/users/me', data);

export const sendCurrentUserVerifyCode = (data: UserVerifyCodeReq) =>
  http.post<unknown, NoDataResp>('/users/me/verify-code/send', data);

export const updatePassword = (data: UpdatePasswordReq) => 
  http.put<unknown, NoDataResp>('/users/me/password', data);

// Admin
export const getUsers = (params?: PageParams) => 
  http.get<unknown, BaseResp<PaginatedResp<User>>>('/admin/users', { params });

export const updateUserStatus = (id: number, status: UserStatus) =>
  http.patch<unknown, NoDataResp>(`/admin/users/${id}/status`, { status } as UpdateUserStatusReq);

export const updateUserRole = (id: number, role: UserRole) =>
  http.patch<unknown, NoDataResp>(`/admin/users/${id}/role`, { role } as UpdateUserRoleReq);

export const getLogs = (params?: OperationLogListParams) =>
  http.get<unknown, BaseResp<PaginatedResp<Log>>>('/admin/logs', { params });
