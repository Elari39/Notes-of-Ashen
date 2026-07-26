import http from '../utils/http';
import type { BaseResp, MediaAsset, NoDataResp, PaginatedResp } from '../types';
import type { PageParams } from '../types/api';

export const MEDIA_UPLOAD_ACCEPT = 'image/jpeg,image/png,image/gif,image/webp,image/avif';

export const getMediaAssets = (params?: PageParams, signal?: AbortSignal) =>
  http.get<unknown, BaseResp<PaginatedResp<MediaAsset>>>('/admin/media', { params, signal });

export const uploadMedia = (file: File, altText = '') => {
  const data = new FormData(); data.append('file', file); data.append('altText', altText);
  return http.post<unknown, BaseResp<MediaAsset>>('/admin/media', data);
};

export const updateMediaAlt = (id: number, altText: string) =>
  http.patch<unknown, BaseResp<MediaAsset>>(`/admin/media/${id}`, { altText });

export const deleteMedia = (id: number) => http.delete<unknown, NoDataResp>(`/admin/media/${id}`);
