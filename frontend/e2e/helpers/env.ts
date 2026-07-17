import process from 'node:process';

const requiredEnv = (name: string): string => {
  const value = process.env[name]?.trim();
  if (!value) {
    throw new Error(`E2E requires ${name}; start the test Compose stack before running Playwright.`);
  }
  return value.replace(/\/+$/, '');
};

export const e2eEnv = {
  webBaseUrl: requiredEnv('E2E_WEB_BASE_URL'),
  apiBaseUrl: requiredEnv('E2E_API_BASE_URL'),
  redisUrl: requiredEnv('E2E_REDIS_URL'),
};

export const apiV1URL = (path: string): string => {
  const normalizedPath = path.startsWith('/') ? path : `/${path}`;
  return `${e2eEnv.apiBaseUrl}/api/v1${normalizedPath}`;
};
