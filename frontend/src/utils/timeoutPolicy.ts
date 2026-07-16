import type { AxiosRequestConfig } from 'axios';

const TIMEOUT_DEFAULT_GET = 10_000;
const TIMEOUT_DEFAULT_WRITE = 30_000;
const TIMEOUT_LONG_RUNNING = 600_000;

const SERVER_MANAGED_AI_PATTERNS: RegExp[] = [
  /\/articles\/ai\/assist(?:\?|$)/,
  /\/admin\/ai\/(?:models|test)(?:\?|$)/,
];

const LONG_RUNNING_PATTERNS: RegExp[] = [
  /\/articles\/import\b/,
  /\/articles\/[^/]+\/export\b/,
  /\/admin\/search\/reindex\b/,
  /\/admin\/backups\/(?:export|restore)\b/,
];

export const resolveDefaultTimeout = (config: AxiosRequestConfig): number => {
  const url = config.url ?? '';
  if (SERVER_MANAGED_AI_PATTERNS.some((pattern) => pattern.test(url))) {
    return 0;
  }
  if (LONG_RUNNING_PATTERNS.some((pattern) => pattern.test(url))) {
    return TIMEOUT_LONG_RUNNING;
  }
  const method = (config.method || 'get').toLowerCase();
  return method === 'get' ? TIMEOUT_DEFAULT_GET : TIMEOUT_DEFAULT_WRITE;
};
