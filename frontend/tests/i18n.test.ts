import assert from 'node:assert/strict';
import test from 'node:test';
import { formatArticleCount, translate } from '../src/i18n.ts';

test('文章数量按语言正确处理英文单复数', () => {
  assert.equal(formatArticleCount('zh', 0), '已收录 0 篇文章');
  assert.equal(formatArticleCount('zh', 1), '已收录 1 篇文章');
  assert.equal(formatArticleCount('en', 0), '0 articles collected');
  assert.equal(formatArticleCount('en', 1), '1 article collected');
  assert.equal(formatArticleCount('en', 2), '2 articles collected');
});

test('首页默认描述和状态文案提供完整中英文翻译', () => {
  assert.equal(translate('zh', 'home.defaultSiteDescription'), '一份以墨为灯、缓慢书写的个人博客。');
  assert.equal(translate('en', 'home.defaultSiteDescription'), 'A personal blog written slowly by the lamp of ink.');
  assert.equal(translate('zh', 'home.featuredRead'), '阅读全文');
  assert.equal(translate('en', 'home.featuredRead'), 'Read article');
  assert.equal(translate('zh', 'home.emptyTitle'), '卷帙未盈。');
  assert.equal(translate('en', 'home.emptyTitle'), 'No articles yet.');
});

test('操作日志筛选和详情文案提供完整中英文翻译', () => {
  assert.equal(translate('zh', 'logs.filteredEmptyTitle'), '没有匹配的日志');
  assert.equal(translate('en', 'logs.filteredEmptyTitle'), 'No matching logs');
  assert.equal(translate('zh', 'logs.event.tokenMismatchLogout'), '异常令牌退出');
  assert.equal(translate('en', 'logs.event.tokenMismatchLogout'), 'Suspicious token logout');
});
