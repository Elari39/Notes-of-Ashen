import http from '../utils/http';
import type { AnalyticsOverview, ArticleAnalytics, ArticleAnalyticsDetail, BaseResp, PaginatedResp } from '../types';
import type { AnalyticsParams } from '../types/api';

export const getAnalyticsOverview = (params?: AnalyticsParams, signal?: AbortSignal) =>
  http.get<unknown, BaseResp<AnalyticsOverview>>('/admin/analytics/overview', { params, signal });
export const getArticleAnalytics = (params?: AnalyticsParams, signal?: AbortSignal) =>
  http.get<unknown, BaseResp<PaginatedResp<ArticleAnalytics>>>('/admin/analytics/articles', { params, signal });
export const getArticleAnalyticsDetail = (id: number, params?: AnalyticsParams, signal?: AbortSignal) =>
  http.get<unknown, BaseResp<ArticleAnalyticsDetail>>(`/admin/analytics/articles/${id}`, { params, signal });
