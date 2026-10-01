import { Buffer } from 'node:buffer';
import {
  expect,
  test,
  type APIResponse,
  type Page,
  type StorageState,
} from '@playwright/test';
import {
  apiV1URL,
  e2eEnv,
} from './helpers/env';
import {
  createIdentity,
  FIXED_ADMIN_IDENTITY,
  loginThroughUI,
  matchesAPIPath,
  preparePage,
  refreshAccessToken,
  registerThroughUI,
  type TestIdentity,
  updateUserRole,
} from './helpers/auth';
import { expireIssuedAccessTokens, readLoginCaptcha, seedRegisterEmailCode, waitForLoginQuota } from './helpers/redis';

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
  let publishedArticle: Article | undefined;

  const requireAdminState = (): StorageState => {
    if (!adminStorageState) {
      throw new Error('Admin browser state was not created by the first-registration test.');
    }
    return adminStorageState;
  };

  test('首次注册管理员并在刷新后恢复会话', async ({ page }) => {
    // 固定凭据：空库注册为 admin；手动复用隔离环境时登录已有管理员。
    // 测试是否通过仍由下面的真实认证与权限断言决定，不自动重试失败用例。
    admin = { ...FIXED_ADMIN_IDENTITY };
    await preparePage(page);

    // registrationEmailCodeRequired=false 仅在"用户表为空且邮箱服务未启用"时成立
    // （首个注册豁免）；为 true 说明管理员已存在（serial 重跑场景）。
    const settingsResponse = page.waitForResponse((response) => matchesAPIPath(response, '/site/settings', 'GET'));
    await page.goto('/register');
    const settings = await successData<{ registrationEmailCodeRequired?: boolean }>(
      await settingsResponse,
      'Site settings lookup',
    );

    const currentUser = settings.registrationEmailCodeRequired === false
      ? await registerThroughUI(page, admin)
      : await loginThroughUI(page, admin);
    expect(currentUser.role).toBe('admin');

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
      await page.getByTestId('article-editor-content').fill('The E2E article body starts here.\n\n```mermaid\nflowchart LR\nA[Audit] --> B[Cleanup]\n```');

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
      publishedArticle = article;
      await expect(page).toHaveURL(/\/admin\/articles$/);
      await expect(page.getByText(title, { exact: true })).toBeVisible();

      await page.goto(`/article/${article.id}`);
      await expect(page.getByRole('heading', { name: title, exact: true })).toBeVisible();
    } finally {
      adminStorageState = await context.storageState();
      await context.close();
    }
  });

  for (const style of ['editorial', 'soft-brutalism', 'japanese-paper', 'swiss', 'neo-brutalism', 'dark-academia']) {
    for (const mode of ['light', 'dark']) {
      test(`真实文章 Mermaid 对比度：${style}/${mode}`, async ({ page }) => {
        if (!publishedArticle) throw new Error('Missing real article fixture');
        await page.addInitScript(({ style, mode }) => {
          localStorage.setItem('notesOfAshen.themeStyle', style);
          localStorage.setItem('notesOfAshen.theme', mode);
        }, { style, mode });
        await page.goto(`/article/${publishedArticle.id}`);
        await expect(page.locator('.article-mermaid-panel')).toHaveAttribute('aria-busy', 'false');
        await expect(page.locator('.article-mermaid-controls')).toBeVisible();
        const colors = await page.locator('.article-mermaid-stage svg .node').filter({ hasText: 'Audit' }).evaluate((node) => {
          const label = node.querySelector('.nodeLabel p');
          const rect = node.querySelector('rect');
          if (!label || !rect) throw new Error('Rendered Mermaid label/background missing');
          const foreground = getComputedStyle(label).color;
          const background = getComputedStyle(rect).fill;
          const luminance = (color: string) => {
            const values = color.match(/[\d.]+/g)?.slice(0, 3).map(Number);
            if (!values || values.length !== 3) throw new Error(`Invalid color: ${color}`);
            return values.map((value) => value / 255).map((v) => v <= 0.04045 ? v / 12.92 : ((v + 0.055) / 1.055) ** 2.4)
              .reduce((sum, value, i) => sum + value * [0.2126, 0.7152, 0.0722][i], 0);
          };
          const a = luminance(foreground);
          const b = luminance(background);
          return { foreground, background, contrast: (Math.max(a, b) + 0.05) / (Math.min(a, b) + 0.05) };
        });
        await test.info().attach('mermaid-contrast', { body: JSON.stringify(colors), contentType: 'application/json' });
        expect(colors.contrast).toBeGreaterThanOrEqual(4.5);
      });
    }
  }

  for (const style of ['neo-brutalism', 'dark-academia']) {
    for (const mode of ['light', 'dark']) {
      test(`新主题后台导航与页面布局：${style}/${mode}`, async ({ browser }, testInfo) => {
        const context = await browser.newContext({
          storageState: requireAdminState(),
          viewport: testInfo.project.use.viewport,
          isMobile: testInfo.project.use.isMobile,
          hasTouch: testInfo.project.use.hasTouch,
          deviceScaleFactor: testInfo.project.use.deviceScaleFactor,
          userAgent: testInfo.project.use.userAgent,
        });
        await context.addInitScript(({ style, mode }) => {
          localStorage.setItem('notesOfAshen.themeStyle', style);
          localStorage.setItem('notesOfAshen.theme', mode);
          localStorage.setItem('notesOfAshen.language', 'en');
        }, { style, mode });
        const page = await context.newPage();
        const errors: string[] = [];
        page.on('pageerror', (error) => errors.push(error.message));
        try {
          for (const route of ['articles', 'categories', 'tags', 'media', 'dashboard', 'analytics', 'users', 'logs', 'settings', 'ai-settings', 'rag-settings', 'system']) {
            await page.goto(`/admin/${route}`);
            await expect(page.locator('.admin-workspace')).toBeVisible();
            await expect(page.locator('#admin-group-content')).toBeVisible();
            await expect(page.locator('#admin-group-system')).toBeVisible();
            await expect(page.locator('.admin-workspace h1')).toBeVisible();
            await expect.poll(() => page.evaluate(() => document.documentElement.scrollWidth - document.documentElement.clientWidth)).toBeLessThanOrEqual(1);
            await page.screenshot({ path: testInfo.outputPath(`${route}.png`), fullPage: true });
            if (route === 'settings') {
              const toggle = page.getByRole('switch', { name: 'Continue Exploring Module', exact: true });
              await expect(toggle).toBeEnabled();
              const initial = await toggle.getAttribute('aria-checked');
              for (const expected of [initial === 'true' ? 'false' : 'true', initial]) {
                await toggle.click();
                const saved = page.waitForResponse((response) => matchesAPIPath(response, '/admin/site/settings', 'PUT'));
                await page.getByRole('button', { name: 'Save', exact: true }).click();
                await successData(await saved, 'Save site presentation settings');
                await page.reload();
                await expect(toggle).toHaveAttribute('aria-checked', expected || 'false');
              }
            }
          }
          expect(errors).toEqual([]);
        } finally {
          adminStorageState = await context.storageState();
          await context.close();
        }
      });
    }
  }

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

  test('移动端公共页面与后台核心表格可触控使用', async ({ browser, page }) => {
    test.skip(!test.info().project.name.startsWith('mobile-'), '仅在移动端项目验证移动布局与触控路径。');

    if (!publishedArticle) {
      throw new Error('The mobile layout test requires the article created by the preceding article test.');
    }

    const assertNoHorizontalOverflow = async (targetPage: Page): Promise<void> => {
      await expect.poll(
        () => targetPage.evaluate(() => Math.max(document.documentElement.scrollWidth, document.body.scrollWidth) - window.innerWidth),
        { message: '页面不应出现横向滚动条' },
      ).toBeLessThanOrEqual(1);
    };

    await preparePage(page);
    await page.goto('/');
    await expect(page.getByRole('main')).toBeVisible();
    await assertNoHorizontalOverflow(page);

    const menuButton = page.getByRole('button', { name: 'Open menu', exact: true });
    await menuButton.tap();
    const mobileNavigation = page.getByRole('navigation', { name: 'Open menu', exact: true });
    await expect(mobileNavigation).toBeVisible();
    await mobileNavigation.getByRole('link', { name: 'Search', exact: true }).tap();
    await expect(page).toHaveURL(/\/search$/);
    await expect(page.getByPlaceholder('Title, summary, or body', { exact: true })).toBeVisible();
    await assertNoHorizontalOverflow(page);

    await page.goto(`/article/${publishedArticle.id}`);
    await expect(page.getByRole('heading', { name: publishedArticle.title, exact: true })).toBeVisible();
    await assertNoHorizontalOverflow(page);

    await page.goto('/login');
    await expect(page.getByLabel('Account or email', { exact: true })).toBeVisible();
    await assertNoHorizontalOverflow(page);

    const adminContext = await browser.newContext();
    await adminContext.addInitScript(() => {
      window.localStorage.setItem('notesOfAshen.language', 'en');
    });
    const adminPage = await adminContext.newPage();
    try {
      // 上游 API 刷新测试会轮换 Refresh Cookie；通过真实移动端登录取得当前有效会话。
      await loginThroughUI(adminPage, admin);
      await adminPage.goto('/admin/articles');
      await expect(adminPage.getByRole('heading', { name: 'Articles', exact: true })).toBeVisible();
      await expect(adminPage.locator('table')).toBeVisible();
      await assertNoHorizontalOverflow(adminPage);
    } finally {
      await adminContext.close();
    }
  });

  test('Access Token 失效时注销仍撤销 Refresh Cookie', async ({ browser }) => {
    const context = await browser.newContext();
    const page = await context.newPage();
    try {
      await preparePage(page);
      await loginThroughUI(page, admin);

      const logoutResponse = await context.request.post(apiV1URL('/auth/logout'), {
        headers: { Authorization: 'Bearer expired-access-token' },
        data: {},
      });
      const logoutPayload = await logoutResponse.json() as ApiEnvelope<unknown>;
      expect(logoutResponse.status()).toBe(200);
      expect(logoutPayload.code).toBe(0);
      expect(logoutResponse.headers()['set-cookie']).toMatch(/noa_refresh_token=.*Max-Age=0/i);

      const refreshResponse = await context.request.post(apiV1URL('/auth/refresh'), { data: {} });
      expect(refreshResponse.status()).toBe(401);
    } finally {
      await context.close();
    }
  });

  test('真实会话失效后登录返回原后台路径', async ({ page }) => {
    await waitForLoginQuota(2);
    await preparePage(page);
    await loginThroughUI(page, admin);
    await page.goto('/admin/articles');
    await expect(page.getByRole('heading', { name: 'Articles', exact: true })).toBeVisible();
    const logout = await page.context().request.post(apiV1URL('/auth/logout'), { data: {} });
    expect(logout.status()).toBe(200);
    const cutoff = await expireIssuedAccessTokens();
    const captcha = page.waitForResponse((response) => matchesAPIPath(response, '/auth/captcha', 'POST'));
    await page.getByRole('link', { name: 'Users', exact: true }).click();
    await expect(page).toHaveURL(/\/login$/);
    expect(await page.evaluate(() => window.history.state.usr.from)).toBe('/admin/users');
    const { captchaId } = await successData<{ captchaId: string }>(await captcha, 'Expired-session captcha');
    await page.getByLabel('Account or email', { exact: true }).fill(admin.account);
    await page.getByLabel('Password', { exact: true }).fill(admin.password);
    await page.getByLabel('Captcha', { exact: true }).fill(await readLoginCaptcha(captchaId));
    // 新令牌必须晚于服务端的失效秒，不缩短失效窗口或修改认证检查。
    await page.waitForTimeout(Math.max(0, (cutoff + 1) * 1000 - Date.now()));
    const login = page.waitForResponse((response) => matchesAPIPath(response, '/auth/login', 'POST'));
    await page.getByRole('button', { name: 'Sign In', exact: true }).click();
    await successData(await login, 'Login after session expiry');
    await expect(page).toHaveURL(/\/admin\/users$/);
    await expect(page.getByRole('heading', { name: 'Users', exact: true })).toBeVisible();
    adminStorageState = await page.context().storageState();
  });

  test('真实删除文章后，读者离线刷新不能恢复已撤回内容', async ({ browser, request }) => {
    if (!publishedArticle) throw new Error('Missing real article fixture');
    const reader = await browser.newContext();
    const page = await reader.newPage();
    try {
      await preparePage(page);
      await page.goto(`/article/${publishedArticle.id}`);
      await page.evaluate(() => navigator.serviceWorker.ready);
      await page.reload();
      await expect(page.getByRole('heading', { name: publishedArticle.title, exact: true })).toBeVisible();
      const detailPath = `/api/v1/articles/${publishedArticle.id}`;
      const hasCachedDetail = () => page.evaluate(async (path) => {
        for (const name of await caches.keys()) {
          if (name.endsWith(':articles') && await (await caches.open(name)).match(path)) return true;
        }
        return false;
      }, detailPath);
      await expect.poll(hasCachedDetail).toBe(true);
      const token = await refreshAccessToken(request, requireAdminState());
      const deleted = await request.delete(apiV1URL(`/articles/${publishedArticle.id}`), { headers: { Authorization: `Bearer ${token}` } });
      expect(deleted.status()).toBe(200);
      expect((await deleted.json() as ApiEnvelope<unknown>).code).toBe(0);
      const detail = page.waitForResponse((response) => matchesAPIPath(response, `/articles/${publishedArticle!.id}`, 'GET'));
      await page.reload();
      expect((await detail).status()).toBe(404);
      await expect.poll(hasCachedDetail).toBe(false);
      await reader.setOffline(true);
      await page.reload();
      await expect(page.getByRole('main')).toBeVisible();
      await expect(page.getByRole('heading', { name: publishedArticle.title, exact: true })).toHaveCount(0);
      expect(await hasCachedDetail()).toBe(false);
    } finally {
      await reader.close();
    }
  });
});
