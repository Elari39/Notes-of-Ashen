import http from '../utils/http';
import type { BaseResp, SiteSettings } from '../types';
import type { UpdateSiteSettingsReq } from '../types/api';

export const getSiteSettings = () =>
  http.get<unknown, BaseResp<SiteSettings>>('/site/settings');

export const updateSiteSettings = (data: UpdateSiteSettingsReq) =>
  http.put<unknown, BaseResp<SiteSettings>>('/admin/site/settings', data);
