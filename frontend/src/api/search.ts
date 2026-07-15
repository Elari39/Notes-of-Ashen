import http from '../utils/http';
import type { BaseResp, SearchSuggestion } from '../types';

export const getSearchSuggestions = (q: string, signal?: AbortSignal) =>
  http.get<unknown, BaseResp<{ items: SearchSuggestion[] }>>('/search/suggestions', { params: { q, limit: 8 }, signal });
