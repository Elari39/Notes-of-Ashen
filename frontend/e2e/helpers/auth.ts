import { expect, type APIRequestContext, type APIResponse, type Page, type StorageState } from '@playwright/test';
import { e2eEnv, apiV1URL } from './env';
import { readLoginCaptcha, readRegisterCaptcha } from './redis';

export type TestIdentity = {
  account: string;
  email: string;
  password: string;
};

type ApiEnvelope<T> = {
  code: number;
  message?: string;
  data?: T;
};

export type CurrentUser = {
  id: number;
  account: string;
  role: string;
};

const refreshCookieName = 'noa_refresh_token';

const apiPathMatches = (response: APIResponse, path: string, method?: string): boolean => {
  const pathname = new URL(response.url()).pathname;
  return pathname === `/api/v1${path}` && (!method || response.request().method() === method);
};

const readEnvelope = async <T>(response: APIResponse, operation: string): Promise<T> => {
  const payload = await response.json() as ApiEnvelope<T>;
  if (!response.ok() || payload.code !== 0 || !payload.data) {
    throw new Error(`${operation} failed: HTTP ${response.status()} (${payload.message || 'unexpected API response'}).`);
  }
  return payload.data;
};

const assertSuccess = async (response: APIResponse, operation: string): Promise<void> => {
  const payload = await response.json() as ApiEnvelope<unknown>;
  if (!response.ok() || payload.code !== 0) {
    throw new Error(`${operation} failed: HTTP ${response.status()} (${payload.message || 'unexpected API response'}).`);
  }
};

const refreshCookieFromState = (storageState: StorageState): string => {
  const cookie = storageState.cookies.find((item) => item.name === refreshCookieName);
  if (!cookie?.value) {
    throw new Error('No refresh cookie was found in the browser storage state.');
  }
  return cookie.value;
};

export const preparePage = async (page: Page): Promise<void> => {
  await page.addInitScript(() => {
    window.localStorage.setItem('notesOfAshen.language', 'en');
  });
};

export const createIdentity = (prefix: string): TestIdentity => {
  const suffix = `${Date.now()}${Math.floor(Math.random() * 100_000)}`;
  const account = `${prefix}_${suffix}`.slice(0, 64);
  return {
    account,
    email: `${account}@example.test`,
    password: `NoaE2E!${suffix}`,
  };
};

const captchaIDFromResponse = async (response: APIResponse): Promise<string> => {
  const data = await readEnvelope<{ captchaId?: string }>(response, 'Captcha request');
  if (!data.captchaId) {
    throw new Error('Captcha response did not contain captchaId.');
  }
  return data.captchaId;
};

export const loginThroughUI = async (page: Page, identity: TestIdentity): Promise<void> => {
  const captchaResponse = page.waitForResponse((response) => apiPathMatches(response, '/auth/captcha', 'POST'));
  await page.goto('/login');
  const captchaID = await captchaIDFromResponse(await captchaResponse);
  const captchaCode = await readLoginCaptcha(captchaID);

  await page.getByLabel('Account or email', { exact: true }).fill(identity.account);
  await page.getByLabel('Password', { exact: true }).fill(identity.password);
  await page.getByLabel('Captcha', { exact: true }).fill(captchaCode);
  const currentUserResponse = page.waitForResponse((response) => apiPathMatches(response, '/users/me', 'GET'));
  await page.getByRole('button', { name: 'Sign In', exact: true }).click();
  await readEnvelope<CurrentUser>(await currentUserResponse, 'Login user lookup');
  await page.waitForURL((url) => url.pathname === '/');
};

export const registerThroughUI = async (
  page: Page,
  identity: TestIdentity,
  emailCode?: string,
): Promise<CurrentUser> => {
  const settingsResponse = page.waitForResponse((response) => apiPathMatches(response, '/site/settings', 'GET'));
  const registerCaptchaResponse = emailCode
    ? page.waitForResponse((response) => apiPathMatches(response, '/auth/captcha', 'POST'))
    : undefined;
  await page.goto('/register');
  await readEnvelope<Record<string, unknown>>(await settingsResponse, 'Site settings lookup');

  await page.getByLabel('Account (3-64 chars)', { exact: true }).fill(identity.account);
  await page.getByLabel('Email', { exact: true }).fill(identity.email);
  await page.getByLabel('Password (at least 8 chars)', { exact: true }).fill(identity.password);
  await page.getByLabel('Confirm password', { exact: true }).fill(identity.password);
  if (emailCode) {
    if (!registerCaptchaResponse) {
      throw new Error('Registration captcha request was not prepared for a verification-code registration.');
    }
    const captchaID = await captchaIDFromResponse(await registerCaptchaResponse);
    const captchaCode = await readRegisterCaptcha(captchaID);
    await page.getByLabel('Captcha', { exact: true }).fill(captchaCode);
    const emailCodeInput = page.getByTestId('register-email-code');
    await emailCodeInput.fill(emailCode);
    await expect(emailCodeInput).toHaveValue(emailCode);
  } else {
    await page.getByTestId('register-email-code').waitFor({ state: 'hidden' });
  }

  const registerResponse = page.waitForResponse((response) => apiPathMatches(response, '/auth/register', 'POST'));
  const currentUserResponse = page.waitForResponse((response) => apiPathMatches(response, '/users/me', 'GET'));
  await page.getByRole('button', { name: 'Register', exact: true }).click();
  await readEnvelope<{ accessToken?: string }>(await registerResponse, 'Registration request');
  const user = await readEnvelope<CurrentUser>(await currentUserResponse, 'Registered user lookup');
  await page.waitForURL((url) => url.pathname === '/');
  return user;
};

export const refreshAccessToken = async (request: APIRequestContext, storageState: StorageState): Promise<string> => {
  const response = await request.post(apiV1URL('/auth/refresh'), {
    headers: {
      Cookie: `${refreshCookieName}=${refreshCookieFromState(storageState)}`,
    },
    data: {},
  });
  const data = await readEnvelope<{ accessToken?: string }>(response, 'Refresh token request');
  if (!data.accessToken) {
    throw new Error('Refresh token response did not contain accessToken.');
  }
  return data.accessToken;
};

export const updateUserRole = async (
  request: APIRequestContext,
  accessToken: string,
  userID: number,
  role: 'editor' | 'admin' | 'user',
): Promise<void> => {
  const response = await request.patch(apiV1URL(`/admin/users/${userID}/role`), {
    headers: {
      Authorization: `Bearer ${accessToken}`,
    },
    data: { role },
  });
  await assertSuccess(response, 'Admin role update');
};

export const webURL = (path: string): string => {
  return new URL(path, `${e2eEnv.webBaseUrl}/`).toString();
};

export const matchesAPIPath = apiPathMatches;
