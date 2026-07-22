import http from '../utils/http';
import { BaseResp, NoDataResp, TokenPair } from '../types';
import {
  CaptchaReq,
  CaptchaResp,
  LoginReq,
  RefreshReq,
  RegisterReq,
  ResetPasswordReq,
  SendVerifyCodeReq,
} from '../types/api';

export const createCaptcha = (data: CaptchaReq) =>
  http.post<unknown, BaseResp<CaptchaResp>>('/auth/captcha', data);

export const sendVerifyCode = (data: SendVerifyCodeReq) =>
  http.post<unknown, NoDataResp>('/auth/verify-code/send', data);

export const login = (data: LoginReq) => 
  http.post<unknown, BaseResp<TokenPair>>('/auth/login', data);

export const register = (data: RegisterReq) => 
  http.post<unknown, BaseResp<TokenPair>>('/auth/register', data);

export const resetPassword = (data: ResetPasswordReq) =>
  http.post<unknown, NoDataResp>('/auth/password/reset', data);

export const refresh = (data: RefreshReq) => 
  http.post<unknown, BaseResp<TokenPair>>('/auth/refresh', data);

export const logout = (data: RefreshReq) => 
  http.post<unknown, NoDataResp>('/auth/logout', data);
