import http from '../utils/http';
import { BaseResp, SiteSettings } from '../types';
import { UpdateSiteSettingsReq } from '../types/api';

export const getSiteSettings = () =>
  http.get<unknown, BaseResp<SiteSettings>>('/site/settings');

export const updateSiteSettings = (data: UpdateSiteSettingsReq) =>
  http.put<unknown, BaseResp<SiteSettings>>('/admin/site/settings', data);
