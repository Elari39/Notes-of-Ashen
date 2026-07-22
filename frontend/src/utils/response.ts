import type { BaseResp, NoDataResp } from '../types';

type ApiSuccessResponse = BaseResp<unknown> | NoDataResp;

const isRecord = (value: unknown): value is Record<string, unknown> => (
  typeof value === 'object' && value !== null
);

// 后端成功响应允许两种稳定形态：查询/写入结果带 data，mutation NoData 不带 data。
// 错误响应即使省略 data，也不能被这里误判为成功。
export const isSuccessResponse = (value: unknown): value is ApiSuccessResponse => {
  if (!isRecord(value) || value.code !== 0 || typeof value.message !== 'string') {
    return false;
  }
  return !Object.prototype.hasOwnProperty.call(value, 'data') || typeof value.data !== 'undefined';
};

export const isNoDataResponse = (value: unknown): value is NoDataResp => (
  isSuccessResponse(value) && !Object.prototype.hasOwnProperty.call(value, 'data')
);
