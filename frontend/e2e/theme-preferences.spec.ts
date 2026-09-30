import { expect, test, type Page } from '@playwright/test';
import type { Article, HomeArticleLayout, SiteSettings } from '../src/types';

// 固定公共接口数据以覆盖长标题、代码块和失败态；真正的后端链路由 critical-path.spec.ts 覆盖。
// 所有 API 请求都在浏览器中拦截，不写入本地预览站点的数据。
test.use({ serviceWorkers: 'block' });

const article: Article = {
  id: 1, authorId: 1, title: '当 Go 服务在容器里悄悄变慢：一次跨越 Nginx、Compose 与 MySQL 连接池的排查手记',
  slug: 'theme-reading', summary: '验证长标题、正文与代码在不同风格下依然清晰可读。',
  content: '## 排查记录\n\n正文应该保持清晰的阅读层级。\n\n```go\nfmt.Println("readable code")\n```\n\n> 留下值得记住的细节。',
  coverUrl: '', status: 'published', viewCount: 1, likeCount: 0, wordCount: 40,
  readingTimeMinutes: 1, isPinned: true, displayPriority: 0,
  seoTitle: '', seoDescription: '', seoKeywords: '',
  createdAt: '2026-09-27T12:00:00Z', updatedAt: '2026-09-27T12:00:00Z',
};
const themes = [
  { id: 'editorial', label: '原有刊物', light: '#cc785c', dark: '#d68a70' },
  { id: 'soft-brutalism', label: '柔和新粗野', light: '#85412b', dark: '#edaa80' },
  { id: 'japanese-paper', label: '日式纸本', light: '#a94335', dark: '#e29380' },
  { id: 'swiss', label: '瑞士平面', light: '#b8371e', dark: '#ff987e' },
] as const;

const installFixtures = async (page: Page) => {
  const state: { mode: 'ready' | 'empty' | 'error'; layout: HomeArticleLayout; release?: Promise<void> } = {
    mode: 'ready', layout: 'standard',
  };
  await page.route('**/api/v1/**', async (route) => {
    const path = new URL(route.request().url()).pathname.replace('/api/v1', '');
    const ok = (data: unknown) => route.fulfill({ json: { code: 0, message: 'success', data } });
    if (path === '/auth/refresh') {
      return route.fulfill({ status: 401, json: { code: 401, message: 'Not signed in' } });
    }
    if (path === '/traffic/visit') return ok(undefined);
    if (path === '/site/settings') {
      const settings: SiteSettings = {
        registrationEnabled: false, registrationEmailCodeRequired: false,
        homeArticleLayout: state.layout, homeCtaHidden: false,
        siteTitle: '', siteDescription: '', siteKeywords: '', siteBaseUrl: '',
        projectsPageEnabled: false, projectsNavHidden: false,
      };
      return ok(settings);
    }
    if (path === '/articles') {
      if (state.release) await state.release;
      if (state.mode === 'error') {
        return route.fulfill({ status: 503, json: { code: 503, message: '主题测试：文章暂不可用' } });
      }
      const items = state.mode === 'empty' ? [] : [article];
      return ok({ items, total: items.length, page: 1, size: 10 });
    }
    if (path === '/articles/1') return ok(article);
    if (path === '/articles/1/context') return ok({ related: [] });
    if (path === '/categories' || path === '/tags') return ok({ items: [], total: 0 });
    if (path === '/search/suggestions') return ok({ items: [] });
    throw new Error(`主题测试未定义的 API：${route.request().method()} ${path}`);
  });
  return state;
};

const openPreferences = async (page: Page) => {
  const menu = page.getByRole('button', { name: '打开菜单', exact: true });
  if (await menu.isVisible()) await menu.click();
  await page.getByRole('button', { name: '打开语言和主题设置', exact: true }).click();
  const panel = page.getByRole('dialog');
  await expect(panel).toBeVisible();
  return panel;
};

const closePreferences = async (page: Page) => {
  await page.keyboard.press('Escape');
  await expect(page.getByRole('dialog', { name: '语言与主题' })).not.toBeVisible();
  const closeMenu = page.getByRole('button', { name: '关闭菜单', exact: true });
  if (await closeMenu.isVisible()) await closeMenu.click();
};

const expectNoOverflow = async (page: Page) => {
  await expect.poll(() => page.evaluate(() => document.documentElement.scrollWidth - document.documentElement.clientWidth)).toBeLessThanOrEqual(1);
};

test.beforeEach(async ({ page }) => {
  await page.emulateMedia({ colorScheme: 'light', reducedMotion: 'reduce' });
});

for (const theme of themes) {
  test(`${theme.label}：明暗切换、刷新保留与恢复原主题`, async ({ page }) => {
    await installFixtures(page);
    await page.goto('/');
    for (const mode of ['light', 'dark'] as const) {
      const panel = await openPreferences(page);
      await panel.getByRole('button', { name: theme.label, exact: true }).click();
      await panel.getByRole('button', { name: mode === 'light' ? '切换为浅色模式' : '切换为深色模式' }).click();
      await expect(panel.getByRole('button', { name: theme.label, exact: true })).toHaveAttribute('aria-pressed', 'true');
      await expect(page.locator('html')).toHaveAttribute('data-style', theme.id);
      await expect(page.locator('html')).toHaveAttribute('data-theme', mode);
      await expect.poll(() => page.locator('html').evaluate((el) => getComputedStyle(el).getPropertyValue('--ochre').trim())).toBe(theme[mode]);
      await expect(panel.getByLabel('主题色', { exact: true })).toHaveValue(theme[mode]);
      await expect(page.locator('meta[name="theme-color"]')).toHaveAttribute('content', theme[mode]);
      await closePreferences(page);
      await page.reload();
      await expect(page.locator('html')).toHaveAttribute('data-style', theme.id);
      await expect(page.locator('html')).toHaveAttribute('data-theme', mode);
      await expect(page.locator('.home-feature h2')).toHaveText(article.title);
      await expect(page.locator('#home-hero-title')).toHaveCSS('font-family', theme.id === 'swiss' || theme.id === 'soft-brutalism' ? /Inter/ : /Cormorant Garamond/);
      if (theme.id === 'soft-brutalism') await expect(page.locator('.home-feature')).toHaveCSS('border-top-width', '2px');
      if (theme.id === 'swiss') await expect(page.locator('.home-feature')).toHaveCSS('border-top-width', '6px');
      await expectNoOverflow(page);
    }
    const panel = await openPreferences(page);
    await panel.getByRole('button', { name: '原有刊物', exact: true }).click();
    await expect(page.locator('html')).toHaveAttribute('data-style', 'editorial');
    await expect(panel.getByLabel('主题色', { exact: true })).toHaveValue('#d68a70');
  });
}

test('自定义强调色跨风格与刷新保留，重置回当前风格默认值', async ({ page }) => {
  await installFixtures(page);
  await page.goto('/');
  let panel = await openPreferences(page);
  await panel.getByLabel('主题色', { exact: true }).fill('#446688');
  await expect(page.locator('meta[name="theme-color"]')).toHaveAttribute('content', '#446688');
  for (const theme of themes.slice(1)) {
    await panel.getByRole('button', { name: theme.label, exact: true }).click();
    await expect(panel.getByLabel('主题色', { exact: true })).toHaveValue('#446688');
    await expect.poll(() => page.locator('html').evaluate((el) => el.style.getPropertyValue('--ochre'))).toBe('#446688');
  }
  await panel.getByRole('button', { name: '切换为深色模式' }).click();
  await closePreferences(page);
  await page.reload();
  panel = await openPreferences(page);
  await expect(panel.getByLabel('主题色', { exact: true })).toHaveValue('#446688');
  await panel.getByRole('button', { name: '重置', exact: true }).click();
  await expect(panel.getByLabel('主题色', { exact: true })).toHaveValue('#ff987e');
  await expect(page.locator('meta[name="theme-color"]')).toHaveAttribute('content', '#ff987e');
  await closePreferences(page);
  await page.reload();
  await expect(page.locator('meta[name="theme-color"]')).toHaveAttribute('content', '#ff987e');
});

test('旧偏好与无效风格安全回退，系统明暗不会覆盖显式选择', async ({ page }) => {
  await page.addInitScript(() => {
    localStorage.setItem('notesOfAshen.themeStyle', 'unknown-style');
    localStorage.setItem('notesOfAshen.theme', 'dark');
  });
  await installFixtures(page);
  await page.goto('/');
  await expect(page.locator('html')).toHaveAttribute('data-style', 'editorial');
  await expect(page.locator('html')).toHaveAttribute('data-theme', 'dark');
  const panel = await openPreferences(page);
  await panel.getByRole('button', { name: '柔和新粗野', exact: true }).click();
  await panel.getByRole('button', { name: '跟随系统', exact: true }).click();
  await expect(page.locator('html')).toHaveAttribute('data-theme', 'light');
  await page.emulateMedia({ colorScheme: 'dark' });
  await expect(page.locator('html')).toHaveAttribute('data-theme', 'dark');
  await expect(panel.getByLabel('主题色', { exact: true })).toHaveValue('#edaa80');
  await panel.getByRole('button', { name: '切换为浅色模式' }).click();
  await page.emulateMedia({ colorScheme: 'light' });
  await page.emulateMedia({ colorScheme: 'dark' });
  await expect(page.locator('html')).toHaveAttribute('data-theme', 'light');
  await expect(page.locator('html')).toHaveAttribute('data-style', 'soft-brutalism');
});

test('浏览器存储不可用时仍能切换风格', async ({ page }) => {
  await page.addInitScript(() => {
    for (const key of ['getItem', 'setItem', 'removeItem']) {
      Object.defineProperty(Storage.prototype, key, { value: () => { throw new DOMException('Storage unavailable', 'SecurityError'); } });
    }
  });
  await installFixtures(page);
  const errors: string[] = [];
  page.on('pageerror', (error) => errors.push(error.message));
  await page.goto('/');
  const panel = await openPreferences(page);
  await panel.getByRole('button', { name: '日式纸本', exact: true }).click();
  await expect(page.locator('html')).toHaveAttribute('data-style', 'japanese-paper');
  await panel.getByRole('button', { name: '切换为深色模式' }).click();
  await expect(page.locator('html')).toHaveAttribute('data-theme', 'dark');
  await closePreferences(page);
  await expect(page.locator('.home-feature h2')).toHaveText(article.title);
  expect(errors).toEqual([]);
});

test('320px 窄屏下三种风格的中英文、列表和正文不横向溢出', async ({ page }) => {
  const state = await installFixtures(page);
  await page.setViewportSize({ width: 320, height: 740 });
  for (const theme of themes.slice(1)) {
    for (const layout of ['standard', 'alternating'] as const) {
      state.layout = layout;
      await page.goto('/');
      const panel = await openPreferences(page);
      await panel.getByRole('button', { name: theme.label, exact: true }).click();
      await panel.getByRole('button', { name: 'English', exact: true }).click();
      await expect(page.getByRole('button', { name: theme.id === 'swiss' ? 'Swiss' : theme.id === 'japanese-paper' ? 'Japanese Paper' : 'Soft Brutalism', exact: true })).toBeVisible();
      await expectNoOverflow(page);
      await panel.getByRole('button', { name: '中文', exact: true }).click();
      await closePreferences(page);
      await expect(page.locator('.home-article-card')).toHaveCount(1);
      await expectNoOverflow(page);
    }
    await page.goto('/article/1');
    await expect(page.locator('.reading-title')).toHaveText(article.title);
    await expect(page.locator('.article-markdown')).toContainText('readable code');
    await expectNoOverflow(page);
    expect(await page.locator('.reading-title').evaluate((el) => parseFloat(getComputedStyle(el).fontSize))).toBeLessThanOrEqual(36);
    await page.goto('/archive');
    await expect(page.locator('#archive-history-title')).toBeVisible();
    await expectNoOverflow(page);
    await page.goto('/search');
    await expect(page.getByRole('combobox')).toBeVisible();
    await expectNoOverflow(page);
  }
});

test('三种风格在加载、失败、重试及空状态下均保留可用操作', async ({ page }) => {
  const state = await installFixtures(page);
  for (const theme of themes.slice(1)) {
    let release: () => void = () => undefined;
    state.release = new Promise<void>((resolve) => { release = resolve; });
    state.mode = 'error';
    await page.goto('/');
    await expect(page.locator('.home-feature-skeleton')).toHaveCount(4);
    const panel = await openPreferences(page);
    await panel.getByRole('button', { name: theme.label, exact: true }).click();
    await closePreferences(page);
    release();
    // 未知服务端 message 按项目错误策略显示本地化兜底，不直接回显原始错误。
    await expect(page.locator('.home-feature').getByRole('alert').getByText('文章列表加载失败', { exact: true })).toBeVisible();
    state.mode = 'empty';
    await page.locator('.home-feature').getByRole('button', { name: '重试', exact: true }).click();
    await expect(page.locator('.home-feature')).toContainText('卷帙未盈。');
    await expect(page.locator('.home-feature-skeleton')).toHaveCount(0);
    await expect(page.locator('html')).toHaveAttribute('data-style', theme.id);
    await expectNoOverflow(page);
  }
});

test('新主题的默认文字与常用背景满足 4.5:1 对比度', async ({ page }) => {
  await installFixtures(page);
  await page.goto('/');
  const panel = await openPreferences(page);
  const luminance = (hex: string) => {
    expect(hex).toMatch(/^#(?:[\da-f]{3}|[\da-f]{6})$/i);
    const value = hex.length === 4 ? hex.slice(1).split('').map((c) => c + c).join('') : hex.slice(1);
    const channels = [0, 2, 4].map((offset) => parseInt(value.slice(offset, offset + 2), 16) / 255)
      .map((value) => value <= 0.04045 ? value / 12.92 : ((value + 0.055) / 1.055) ** 2.4);
    return channels[0] * 0.2126 + channels[1] * 0.7152 + channels[2] * 0.0722;
  };
  for (const theme of themes.slice(1)) {
    for (const mode of ['light', 'dark'] as const) {
      await panel.getByRole('button', { name: theme.label, exact: true }).click();
      await panel.getByRole('button', { name: mode === 'light' ? '切换为浅色模式' : '切换为深色模式' }).click();
      const colors = await page.locator('html').evaluate((el) => {
        const styles = getComputedStyle(el);
        return Object.fromEntries(['ink', 'body', 'muted', 'ochre', 'on-accent', 'canvas', 'surface-soft', 'surface-card', 'style-peach', 'style-mint', 'style-lilac']
          .map((key) => [key, styles.getPropertyValue(`--${key}`).trim()]));
      });
      const surfaces = ['canvas', 'surface-soft', 'surface-card'];
      if (theme.id === 'soft-brutalism') surfaces.push('style-peach', 'style-mint', 'style-lilac');
      const pairs = surfaces.flatMap((surface) => ['ink', 'body', 'muted', 'ochre'].map((ink) => [ink, surface]));
      pairs.push(['on-accent', 'ochre']);
      for (const [ink, surface] of pairs) {
        const fg = luminance(colors[ink]);
        const bg = luminance(colors[surface]);
        const ratio = (Math.max(fg, bg) + 0.05) / (Math.min(fg, bg) + 0.05);
        expect(ratio, `${theme.label} ${mode} ${ink}/${surface}`).toBeGreaterThanOrEqual(4.5);
      }
    }
  }
});
