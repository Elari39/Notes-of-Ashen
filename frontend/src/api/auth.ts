import http from '../utils/http';
import { BaseResp, TokenPair } from '../types';
import { LoginReq, RegisterReq, RefreshReq } from '../types/api';

export const login = (data: LoginReq) => 
  http.post<any, BaseResp<TokenPair>>('/auth/login', data);

export const register = (data: RegisterReq) => 
  http.post<any, BaseResp<TokenPair>>('/auth/register', data);

export const refresh = (data: RefreshReq) => 
  http.post<any, BaseResp<TokenPair>>('/auth/refresh', data);

export const logout = (data: RefreshReq) => 
  http.post<any, BaseResp>('/auth/logout', data);
