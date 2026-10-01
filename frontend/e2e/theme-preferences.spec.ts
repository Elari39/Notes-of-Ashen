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
  { id: 'neo-brutalism', label: '经典新粗野', light: '#183db5', dark: '#a9c1ff' },
  { id: 'dark-academia', label: '暗色书房', light: '#704719', dark: '#ddbd80' },
] as const;

const installFixtures = async (page: Page) => {
  const state: { mode: 'ready' | 'empty' | 'error'; layout: HomeArticleLayout; release?: Promise<void>; items: Article[] } = {
    mode: 'ready', layout: 'standard', items: [article, { ...article, id: 2, title: '另一篇札记', isPinned: false }],
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
      const all = state.mode === 'empty' ? [] : state.items;
      const currentPage = Number(new URL(route.request().url()).searchParams.get('page') || 1);
      const items = all.slice((currentPage - 1) * 10, currentPage * 10);
      return ok({ items, total: all.length, page: currentPage, size: 10 });
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
  // The preference store can set html attributes before React mounts the mobile navigation.
  await expect(page.getByRole('banner')).toBeVisible();
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
      await expect(page.locator('#home-hero-title')).toHaveCSS('font-family', theme.id === 'swiss' || theme.id === 'soft-brutalism' || theme.id === 'neo-brutalism' ? /Inter/ : /Cormorant Garamond/);
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
  for (const theme of themes) {
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
  await expect(panel.getByLabel('主题色', { exact: true })).toHaveValue('#ddbd80');
  await expect(page.locator('meta[name="theme-color"]')).toHaveAttribute('content', '#ddbd80');
  await closePreferences(page);
  await page.reload();
  await expect(page.locator('meta[name="theme-color"]')).toHaveAttribute('content', '#ddbd80');
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

for (const theme of themes) {
  test(`320px 窄屏 ${theme.label} 的中英文、列表和正文不横向溢出`, async ({ page }) => {
    const state = await installFixtures(page);
    await page.setViewportSize({ width: 320, height: 740 });
    for (const layout of ['standard', 'alternating'] as const) {
      state.layout = layout;
      await page.goto('/');
      const panel = await openPreferences(page);
      await panel.getByRole('button', { name: theme.label, exact: true }).click();
      await panel.getByRole('button', { name: 'English', exact: true }).click();
      await expect(page.getByRole('button', { name: ({ swiss: 'Swiss', 'japanese-paper': 'Japanese Paper', 'soft-brutalism': 'Soft Brutalism', 'neo-brutalism': 'Neo-brutalism', 'dark-academia': 'Dark Academia', editorial: 'Editorial' })[theme.id], exact: true })).toBeVisible();
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
  });
}

test('全部风格在加载、失败、重试及空状态下均保留可用操作', async ({ page }) => {
  const state = await installFixtures(page);
  for (const theme of themes) {
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
  for (const theme of themes.filter((theme) => theme.id !== 'editorial')) {
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


test('首页主推去重，单篇不显示空列表，分页与筛选保留完整文章', async ({ page }) => {
  const state = await installFixtures(page);
  state.items = [article];
  await page.goto('/');
  await expect(page.locator('.home-feature h2')).toHaveText(article.title);
  await expect(page.locator('.home-article-card')).toHaveCount(0);
  await expect(page.locator('#latest-notes')).toHaveCount(0);
  for (const layout of ['standard', 'alternating'] as const) {
    state.layout = layout;
    state.items = Array.from({ length: 11 }, (_, index) => ({ ...article, id: index + 1, title: `文章 ${index + 1}` }));
    await page.goto('/');
    await expect(page.locator('.home-feature h2')).toHaveText('文章 1');
    await expect(page.locator('.home-article-card')).toHaveCount(9);
    await expect(page.locator('.home-article-card').getByRole('heading', { name: '文章 1', exact: true })).toHaveCount(0);
    await page.goto('/?page=2');
    await expect(page.locator('.home-feature')).toHaveCount(0);
    await expect(page.locator('.home-article-card')).toHaveCount(1);
    await expect(page.locator('.home-article-card h3')).toHaveText('文章 11');
    await page.goto('/?categoryId=1');
    await expect(page.locator('.home-feature')).toHaveCount(0);
    await expect(page.locator('.home-article-card')).toHaveCount(10);
    await page.goBack();
    await expect(page.locator('.home-article-card h3')).toHaveText('文章 11');
  }
});

for (const theme of themes) {
  test(`坏封面降级、桌面和平板布局：${theme.label}`, async ({ page }, testInfo) => {
    const state = await installFixtures(page);
    state.items[1].coverUrl = 'https://fixtures.notes.test/broken-cover.png';
    let failedCoverRequests = 0;
  await page.route('**/broken-cover.png', (route) => {
    failedCoverRequests += 1;
    return route.fulfill({ status: 404, body: '' });
  });
    for (const mode of ['light', 'dark']) {
      await page.goto('/');
      const panel = await openPreferences(page);
      await panel.getByRole('button', { name: theme.label, exact: true }).click();
      await panel.getByRole('button', { name: mode === 'light' ? '切换为浅色模式' : '切换为深色模式' }).click();
      await closePreferences(page);
      for (const width of [768, 1440]) {
        await page.setViewportSize({ width, height: 1000 });
        await page.locator('.home-article-card').scrollIntoViewIfNeeded();
        await expect.poll(() => failedCoverRequests).toBeGreaterThan(0);
        await expect(page.locator('.home-article-card img')).toHaveCount(0);
        await expect(page.locator('.home-article-card h3')).toHaveText('另一篇札记');
        await expectNoOverflow(page);
        await page.evaluate(() => window.scrollTo({ top: 0, behavior: 'instant' }));
        await expect.poll(() => page.evaluate(() => window.scrollY)).toBe(0);
        await page.screenshot({ path: testInfo.outputPath(`${theme.id}-${mode}-${width}.png`), fullPage: true });
      }
    }
  });
}


test('新主题的自定义强调色保持主按钮文字可读，偏好可由键盘退出', async ({ page }) => {
  await installFixtures(page);
  await page.goto('/');
  const panel = await openPreferences(page);
  for (const label of ['经典新粗野', '暗色书房']) {
    await panel.getByRole('button', { name: label, exact: true }).click();
    for (const color of ['#000000', '#ffffff', '#446688']) {
      await panel.getByLabel('主题色', { exact: true }).fill(color);
      const ratio = await page.locator('html').evaluate((el) => {
        const css = getComputedStyle(el);
        const luminance = (hex: string) => {
          const values = [1, 3, 5].map((offset) => parseInt(hex.slice(offset, offset + 2), 16) / 255)
            .map((value) => value <= 0.04045 ? value / 12.92 : ((value + 0.055) / 1.055) ** 2.4);
          return values[0] * 0.2126 + values[1] * 0.7152 + values[2] * 0.0722;
        };
        const fg = luminance(css.getPropertyValue('--on-accent').trim());
        const bg = luminance(css.getPropertyValue('--ochre').trim());
        return (Math.max(fg, bg) + 0.05) / (Math.min(fg, bg) + 0.05);
      });
      expect(ratio).toBeGreaterThanOrEqual(4.5);
    }
  }
  await page.keyboard.press('Escape');
  await expect(page.getByRole('dialog', { name: '语言与主题' })).toHaveCount(0);
  await expect(page.getByRole('button', { name: '打开语言和主题设置', exact: true })).toBeFocused();
});


test('交错布局只按可用封面交替，文字卡片与坏图不打断节奏', async ({ page }) => {
  const state = await installFixtures(page);
  state.layout = 'alternating';
  state.items = [article,
    { ...article, id: 2, title: '左侧封面', coverUrl: 'https://fixtures.notes.test/cover.svg' },
    { ...article, id: 3, title: '文字札记' },
    { ...article, id: 4, title: '坏图札记', coverUrl: 'https://fixtures.notes.test/broken.svg' },
    { ...article, id: 5, title: '右侧封面', coverUrl: 'https://fixtures.notes.test/cover.svg' },
  ];
  await page.route('**/cover.svg', (route) => route.fulfill({ contentType: 'image/svg+xml', body: '<svg xmlns="http://www.w3.org/2000/svg" width="160" height="90"><rect width="160" height="90" fill="#ccbbaa"/></svg>' }));
  await page.route('**/broken.svg', (route) => route.fulfill({ status: 404, body: '' }));
  await page.setViewportSize({ width: 1280, height: 1000 });
  await page.goto('/');
  const left = page.locator('.home-article-card').filter({ has: page.getByRole('heading', { name: '左侧封面', exact: true }) });
  const right = page.locator('.home-article-card').filter({ has: page.getByRole('heading', { name: '右侧封面', exact: true }) });
  await right.scrollIntoViewIfNeeded();
  await expect(page.locator('.home-article-card img')).toHaveCount(2);
  await expect(left).toHaveCSS('flex-direction', 'row');
  await expect(right).toHaveCSS('flex-direction', 'row-reverse');
  await expectNoOverflow(page);
});
