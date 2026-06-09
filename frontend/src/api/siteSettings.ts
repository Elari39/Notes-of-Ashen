import http from '../utils/http';
import type { BaseResp, ProjectsPage, ResumePage, SiteSettings } from '../types';
import type { UpdateProjectsPageReq, UpdateResumePageReq, UpdateSiteSettingsReq } from '../types/api';

export const getSiteSettings = () =>
  http.get<unknown, BaseResp<SiteSettings>>('/site/settings');

export const updateSiteSettings = (data: UpdateSiteSettingsReq) =>
  http.put<unknown, BaseResp<SiteSettings>>('/admin/site/settings', data);

export const getResumePage = () =>
  http.get<unknown, BaseResp<ResumePage>>('/site/resume');

export const getAdminResumePage = () =>
  http.get<unknown, BaseResp<ResumePage>>('/admin/site/resume');

export const updateAdminResumePage = (data: UpdateResumePageReq) =>
  http.put<unknown, BaseResp<ResumePage>>('/admin/site/resume', data);

export const getProjectsPage = () =>
  http.get<unknown, BaseResp<ProjectsPage>>('/site/projects');

export const getAdminProjectsPage = () =>
  http.get<unknown, BaseResp<ProjectsPage>>('/admin/site/projects');

export const updateAdminProjectsPage = (data: UpdateProjectsPageReq) =>
  http.put<unknown, BaseResp<ProjectsPage>>('/admin/site/projects', data);
