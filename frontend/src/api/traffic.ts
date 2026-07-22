import http from '../utils/http';
import type { NoDataResp } from '../types';
import type { TrafficVisitReq } from '../types/api';

export const reportVisit = (data: TrafficVisitReq) =>
  http.post<unknown, NoDataResp>('/traffic/visit', data);
