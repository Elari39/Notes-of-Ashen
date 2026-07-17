import { Buffer } from 'node:buffer';
import {
  expect,
  test,
  type APIResponse,
  type StorageState,
} from '@playwright/test';
import {
  apiV1URL,
  e2eEnv,
} from './helpers/env';
import {
  createIdentity,
  loginThroughUI,
  matchesAPIPath,
  preparePage,
  refreshAccessToken,
  registerThroughUI,
  type TestIdentity,
  updateUserRole,
} from './helpers/auth';
import { seedRegisterEmailCode } from './helpers/redis';

type ApiEnvelope<T> = {
  code: number;
  message?: string;
  data?: T;
};

type MediaAsset = {
  id: number;
  url: string;
  originalName: string;
};

type Article = {
  id: number;
  title: string;
};

const imageFixture = {
  name: 'e2e-image.png',
  mimeType: 'image/png',
  // 1×1 PNG。上传内容可由 Go image.DecodeConfig 真实解码，无需仓库二进制 fixture。
  buffer: Buffer.from('iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAADUlEQVQIHWP4z8DwHwAFgAI/ScL7WAAAAABJRU5ErkJggg==', 'base64'),
};

const successData = async <T>(response: APIResponse, operation: string): Promise<T> => {
  const payload = await response.json() as ApiEnvelope<T>;
  expect(response.ok(), `${operation}: HTTP ${response.status()} ${payload.message || ''}`).toBeTruthy();
  expect(payload.code, `${operation}: ${payload.message || 'unexpected API response'}`).toBe(0);
  expect(payload.data, `${operation}: missing response data`).toBeTruthy();
  return payload.data as T;
};

const storageRefreshCookie = (storageState: StorageState): string => {
  const cookie = storageState.cookies.find((item) => item.name === 'noa_refresh_token');
  expect(cookie?.value).toBeTruthy();
  if (!cookie?.value) {
    throw new Error('The expected refresh cookie is missing from browser storage.');
  }
  return cookie.value;
};

test.describe.serial('真实 Compose 前端关键路径', () => {
  let admin: TestIdentity;
  let adminStorageState: StorageState | undefined;

  const requireAdminState = (): StorageState => {
    if (!adminStorageState) {
      throw new Error('Admin browser state was not created by the first-registration test.');
    }
    return adminStorageState;
  };

  test('首次注册管理员并在刷新后恢复会话', async ({ page }) => {
    admin = createIdentity('e2eadmin');
    await preparePage(page);

    const registeredUser = await registerThroughUI(page, admin);
    expect(registeredUser.role).toBe('admin');

    const firstCookie = storageRefreshCookie(await page.context().storageState());
    const refreshResponse = page.waitForResponse((response) => matchesAPIPath(response, '/auth/refresh', 'POST'));
    await page.reload();
    const refreshed = await successData<{ accessToken: string }>(await refreshResponse, 'Browser refresh session');
    expect(refreshed.accessToken).toBeTruthy();

    adminStorageState = await page.context().storageState();
    expect(storageRefreshCookie(adminStorageState)).not.toBe(firstCookie);
    await page.goto('/admin/users');
    await expect(page.getByRole('heading', { name: 'Users', exact: true })).toBeVisible();
  });

  test('登录表单使用真实 Redis 图形验证码恢复管理员访问', async ({ page }) => {
    await preparePage(page);
    await loginThroughUI(page, admin);
    await page.goto('/admin/articles');
    await expect(page.getByRole('heading', { name: 'Articles', exact: true })).toBeVisible();
    // 进入受保护页后初始化可能再次旋转 Cookie，保存最终状态供后续用例复用。
    adminStorageState = await page.context().storageState();
  });

  test('管理员编辑已发布文章并上传、插入真实媒体', async ({ browser }) => {
    const context = await browser.newContext({ storageState: requireAdminState() });
    await context.addInitScript(() => {
      window.localStorage.setItem('notesOfAshen.language', 'en');
    });
    const page = await context.newPage();
    try {
      await page.goto('/admin/articles');
      // 初始化期间可能存在多个 refresh 请求；以实际受保护页面可用作为会话恢复断言。
      await expect(page.getByRole('heading', { name: 'Articles', exact: true })).toBeVisible();

      const timestamp = Date.now();
      const title = `E2E published article ${timestamp}`;
      const slug = `e2e-published-article-${timestamp}`;
      await page.getByTestId('admin-articles-new').click();
      await expect(page).toHaveURL(/\/admin\/editor\/new$/);
      await page.getByTestId('article-editor-title').fill(title);
      await page.getByTestId('article-editor-slug').fill(slug);
      await page.getByTestId('article-editor-status').selectOption('published');
      await page.getByRole('checkbox', { name: /Generate summary on save/i }).uncheck();
      await page.getByTestId('article-editor-content').fill('The E2E article body starts here.');

      await page.getByTestId('article-editor-media-insert').click();
      await expect(page.getByRole('dialog', { name: 'Media Library', exact: true })).toBeVisible();
      const uploadResponse = page.waitForResponse((response) => matchesAPIPath(response, '/admin/media', 'POST'));
      await page.getByTestId('media-picker-upload-input').setInputFiles(imageFixture);
      const uploadedMedia = await successData<MediaAsset>(await uploadResponse, 'Media upload');
      await page.getByTestId(`media-picker-item-${uploadedMedia.id}`).click();

      expect(await page.getByTestId('article-editor-content').inputValue()).toContain(uploadedMedia.url);
      const mediaResponse = await page.request.get(new URL(uploadedMedia.url, e2eEnv.webBaseUrl).toString());
      expect(mediaResponse.ok()).toBeTruthy();

      const createArticleResponse = page.waitForResponse((response) => matchesAPIPath(response, '/articles', 'POST'));
      await page.getByTestId('article-editor-save').click();
      const article = await successData<Article>(await createArticleResponse, 'Article create');
      await expect(page).toHaveURL(/\/admin\/articles$/);
      await expect(page.getByText(title, { exact: true })).toBeVisible();

      await page.goto(`/article/${article.id}`);
      await expect(page.getByRole('heading', { name: title, exact: true })).toBeVisible();
    } finally {
      adminStorageState = await context.storageState();
      await context.close();
    }
  });

  test('管理员改为 editor 后，editor 只能访问文章后台', async ({ browser, request }) => {
    const editor = createIdentity('e2eeditor');
    const verificationCode = '438219';
    await seedRegisterEmailCode(editor.email, verificationCode);

    const registrationContext = await browser.newContext();
    await registrationContext.addInitScript(() => {
      window.localStorage.setItem('notesOfAshen.language', 'en');
    });
    const registrationPage = await registrationContext.newPage();
    let editorUserID: number | undefined;
    try {
      const editorUser = await registerThroughUI(registrationPage, editor, verificationCode);
      editorUserID = editorUser.id;
    } finally {
      await registrationContext.close();
    }

    if (!editorUserID) {
      throw new Error('The editor registration did not return a user ID.');
    }
    const adminAccessToken = await refreshAccessToken(request, requireAdminState());
    await updateUserRole(request, adminAccessToken, editorUserID, 'editor');

    const editorContext = await browser.newContext();
    await editorContext.addInitScript(() => {
      window.localStorage.setItem('notesOfAshen.language', 'en');
    });
    const editorPage = await editorContext.newPage();
    try {
      await loginThroughUI(editorPage, editor);
      await editorPage.goto('/admin/articles');
      await expect(editorPage.getByRole('heading', { name: 'Articles', exact: true })).toBeVisible();

      await editorPage.goto('/admin/users');
      await expect(editorPage.getByText('You do not have permission to access this page.', { exact: true })).toBeVisible();

      // 独立 API refresh 会旋转该 Cookie，因此前端路由断言必须先完成。
      const editorState = await editorContext.storageState();
      const editorAccessToken = await refreshAccessToken(request, editorState);
      const forbiddenResponse = await request.get(apiV1URL('/admin/users'), {
        headers: { Authorization: `Bearer ${editorAccessToken}` },
      });
      expect(forbiddenResponse.status()).toBe(403);
    } finally {
      await editorContext.close();
    }
  });
});
