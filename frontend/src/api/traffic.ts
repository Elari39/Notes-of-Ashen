import http from '../utils/http';
import type { BaseResp } from '../types';
import type { TrafficVisitReq } from '../types/api';

export const reportVisit = (data: TrafficVisitReq) =>
  http.post<unknown, BaseResp>('/traffic/visit', data);
