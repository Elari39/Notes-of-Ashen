export type { ECharts } from 'echarts/core';

type EChartsRuntime = typeof import('./echartsRuntime');

let runtimePromise: Promise<EChartsRuntime> | undefined;

export const loadECharts = () => {
  runtimePromise ??= import('./echartsRuntime');
  return runtimePromise;
};
