import type { Language } from './store/preferences';

const zh = {
  'brand.name': '灰之札记',
  'brand.nameEn': 'Notes of Ashen',
  'nav.home': '诗集',
  'nav.archive': '归韵',
  'nav.admin': '掌卷',
  'nav.login': '结缘',
  'nav.logout': '辞别',
  'nav.profile': '命牍',
  'toggle.language': 'EN',
  'toggle.languageLabel': '切换为英文',
  'toggle.themeLight': '日',
  'toggle.themeDark': '夜',
  'toggle.themeToLight': '切换为浅色',
  'toggle.themeToDark': '切换为深色',
  'footer.poem': '山高水长，落笔生花。',
  'footer.crafted': '以墨为灯，慢写人间。',

  'common.loadingArticles': '研墨中...',
  'common.loadingArticle': '展卷中...',
  'common.loadingArchive': '寻梦中...',
  'common.loadingAuth': '验印中...',
  'common.emptyArticles': '尚无诗笺。',
  'common.views': '阅',
  'common.backHome': '合卷',
  'common.name': '名称',
  'common.description': '描述',
  'common.action': '操作',
  'common.status': '状态',
  'common.time': '时间',
  'common.save': '保存',
  'common.cancel': '取消',
  'common.edit': '修编',
  'common.delete': '销毁',
  'common.processing': '处理中...',
  'common.saving': '保存中...',
  'common.noCategory': '暂无分类',
  'common.noTag': '暂无标签',

  'pagination.prev': '上一卷',
  'pagination.next': '下一卷',
  'pagination.page': '第 {current} / {total} 卷',

  'home.loadError': '文章列表加载失败',
  'home.searchPlaceholder': '搜索标题、摘要或正文',
  'home.search': '搜索',
  'home.clearFilters': '清空',

  'article.loadError': '文章加载失败',
  'article.missing': '此卷已失。',

  'archive.titleCategories': '分类',
  'archive.titleTags': '标签',
  'archive.loadError': '归档数据加载失败',

  'auth.loginTitle': '结缘',
  'auth.registerTitle': '初遇',
  'auth.accountOrEmail': '账号或邮箱',
  'auth.account': '账号',
  'auth.accountWithHint': '账号 (3-64字)',
  'auth.email': '邮箱',
  'auth.nickname': '昵称',
  'auth.nicknameOptional': '昵称 (可选)',
  'auth.avatarOptional': '真容 (Avatar URL, 可选)',
  'auth.avatarUrl': '真容 (Avatar URL)',
  'auth.password': '密码',
  'auth.passwordWithHint': '密码 (至少6字)',
  'auth.oldPassword': '旧梦 (原密码)',
  'auth.newPassword': '新声 (新密码)',
  'auth.loginSubmit': '入卷',
  'auth.loginSubmitting': '入卷中...',
  'auth.registerSubmit': '结缘',
  'auth.registerSubmitting': '结缘中...',
  'auth.noAccount': '尚无缘分？',
  'auth.goRegister': '去结缘',
  'auth.hasAccount': '已有前缘？',
  'auth.goLogin': '去入卷',
  'auth.loginError': '登录失败，请检查账号密码',
  'auth.registerError': '注册失败，请检查填写信息',

  'profile.title': '命牍 (资料)',
  'profile.accountLabel': '账文',
  'profile.nicknameLabel': '别号',
  'profile.emailLabel': '飞书 (Email)',
  'profile.avatarLabel': '真容 (Avatar URL)',
  'profile.passwordTitle': '秘钥 (密码)',
  'profile.updateProfile': '修簜',
  'profile.updatingProfile': '修簜中...',
  'profile.updatePassword': '更易',
  'profile.updatingPassword': '更易中...',
  'profile.updated': '资料已更新',
  'profile.passwordUpdated': '密码已更新',
  'profile.updateError': '更新失败',

  'protected.forbidden': '非掌卷人，无权入内。',

  'notFound.message': '前尘影事，迷失于此。',
  'notFound.back': '寻路归卷',

  'admin.title': '掌卷',
  'admin.articles': '文章管理',
  'admin.categories': '分类管理',
  'admin.tags': '标签管理',
  'admin.users': '掌印 (用户)',
  'admin.logs': '青史 (日志)',

  'adminArticles.new': '提笔',
  'adminArticles.title': '标题',
  'adminArticles.taxonomy': '分类/标签',
  'adminArticles.publish': '发布',
  'adminArticles.archive': '归档',
  'adminArticles.confirmDelete': '确认删除此篇？',
  'adminArticles.loadError': '文章列表加载失败',
  'adminArticles.statusError': '状态更新失败',
  'adminArticles.deleteError': '删除失败',
  'adminArticles.allStatus': '全部状态',
  'adminArticles.allCategories': '全部分类',
  'adminArticles.allTags': '全部标签',

  'articleEditor.editTitle': '修编',
  'articleEditor.newTitle': '提笔',
  'articleEditor.save': '落笔保存',
  'articleEditor.titlePlaceholder': '标题',
  'articleEditor.slugPlaceholder': 'Slug (路径)',
  'articleEditor.coverPlaceholder': '封面 URL',
  'articleEditor.chooseCategory': '选择分类...',
  'articleEditor.tags': '标签:',
  'articleEditor.summary': '摘要',
  'articleEditor.content': '正文 (Markdown)...',
  'articleEditor.depsError': '分类和标签加载失败',
  'articleEditor.saveError': '保存失败',

  'taxonomy.listCategoryError': '分类列表加载失败',
  'taxonomy.listTagError': '标签列表加载失败',
  'taxonomy.submitError': '操作失败',
  'taxonomy.deleteCategoryError': '删除失败，可能存在关联文章',
  'taxonomy.deleteTagError': '删除失败',
  'taxonomy.confirmDelete': '确认删除？',
  'taxonomy.add': '新增',

  'users.account': '账文',
  'users.nickname': '别号',
  'users.role': '权柄',
  'users.activate': '召回',
  'users.disable': '流放',
  'users.confirmStatus': '确认{action}此人？',
  'users.loadError': '用户列表加载失败',
  'users.actionError': '操作失败',

  'logs.user': '谁人',
  'logs.event': '何事',
  'logs.source': '源头',
  'logs.loadError': '日志列表加载失败',
} as const;

type TranslationKey = keyof typeof zh;

const en: Record<TranslationKey, string> = {
  'brand.name': 'Notes of Ashen',
  'brand.nameEn': 'Notes of Ashen',
  'nav.home': 'Poems',
  'nav.archive': 'Archive',
  'nav.admin': 'Desk',
  'nav.login': 'Sign In',
  'nav.logout': 'Farewell',
  'nav.profile': 'Profile',
  'toggle.language': '中',
  'toggle.languageLabel': 'Switch to Chinese',
  'toggle.themeLight': 'Sun',
  'toggle.themeDark': 'Moon',
  'toggle.themeToLight': 'Switch to light mode',
  'toggle.themeToDark': 'Switch to dark mode',
  'footer.poem': 'The road is long; ink still flowers.',
  'footer.crafted': 'Written slowly by the lamp of ink.',

  'common.loadingArticles': 'Grinding ink...',
  'common.loadingArticle': 'Opening the scroll...',
  'common.loadingArchive': 'Seeking echoes...',
  'common.loadingAuth': 'Reading the seal...',
  'common.emptyArticles': 'No letters have settled here.',
  'common.views': 'Views',
  'common.backHome': 'Close Scroll',
  'common.name': 'Name',
  'common.description': 'Description',
  'common.action': 'Action',
  'common.status': 'Status',
  'common.time': 'Time',
  'common.save': 'Save',
  'common.cancel': 'Cancel',
  'common.edit': 'Edit',
  'common.delete': 'Delete',
  'common.processing': 'Working...',
  'common.saving': 'Saving...',
  'common.noCategory': 'No categories yet',
  'common.noTag': 'No tags yet',

  'pagination.prev': 'Previous',
  'pagination.next': 'Next',
  'pagination.page': 'Page {current} / {total}',

  'home.loadError': 'Failed to load articles',
  'home.searchPlaceholder': 'Search title, summary, or body',
  'home.search': 'Search',
  'home.clearFilters': 'Clear',

  'article.loadError': 'Failed to load article',
  'article.missing': 'This scroll has gone missing.',

  'archive.titleCategories': 'Categories',
  'archive.titleTags': 'Tags',
  'archive.loadError': 'Failed to load archive data',

  'auth.loginTitle': 'Sign In',
  'auth.registerTitle': 'Begin',
  'auth.accountOrEmail': 'Account or email',
  'auth.account': 'Account',
  'auth.accountWithHint': 'Account (3-64 chars)',
  'auth.email': 'Email',
  'auth.nickname': 'Nickname',
  'auth.nicknameOptional': 'Nickname (optional)',
  'auth.avatarOptional': 'Avatar URL (optional)',
  'auth.avatarUrl': 'Avatar URL',
  'auth.password': 'Password',
  'auth.passwordWithHint': 'Password (at least 6 chars)',
  'auth.oldPassword': 'Old password',
  'auth.newPassword': 'New password',
  'auth.loginSubmit': 'Enter',
  'auth.loginSubmitting': 'Entering...',
  'auth.registerSubmit': 'Join',
  'auth.registerSubmitting': 'Joining...',
  'auth.noAccount': 'No bond yet?',
  'auth.goRegister': 'Create one',
  'auth.hasAccount': 'Already known here?',
  'auth.goLogin': 'Enter',
  'auth.loginError': 'Login failed. Please check your account and password',
  'auth.registerError': 'Registration failed. Please check your details',

  'profile.title': 'Profile',
  'profile.accountLabel': 'Account',
  'profile.nicknameLabel': 'Nickname',
  'profile.emailLabel': 'Email',
  'profile.avatarLabel': 'Avatar URL',
  'profile.passwordTitle': 'Password',
  'profile.updateProfile': 'Update',
  'profile.updatingProfile': 'Updating...',
  'profile.updatePassword': 'Change',
  'profile.updatingPassword': 'Changing...',
  'profile.updated': 'Profile updated',
  'profile.passwordUpdated': 'Password updated',
  'profile.updateError': 'Update failed',

  'protected.forbidden': 'Only the keeper may enter.',

  'notFound.message': 'Old shadows wander; this path is lost.',
  'notFound.back': 'Return Home',

  'admin.title': 'Desk',
  'admin.articles': 'Articles',
  'admin.categories': 'Categories',
  'admin.tags': 'Tags',
  'admin.users': 'Users',
  'admin.logs': 'Logs',

  'adminArticles.new': 'Write',
  'adminArticles.title': 'Title',
  'adminArticles.taxonomy': 'Category / Tags',
  'adminArticles.publish': 'Publish',
  'adminArticles.archive': 'Archive',
  'adminArticles.confirmDelete': 'Delete this article?',
  'adminArticles.loadError': 'Failed to load article list',
  'adminArticles.statusError': 'Failed to update status',
  'adminArticles.deleteError': 'Failed to delete',
  'adminArticles.allStatus': 'All statuses',
  'adminArticles.allCategories': 'All categories',
  'adminArticles.allTags': 'All tags',

  'articleEditor.editTitle': 'Edit',
  'articleEditor.newTitle': 'Write',
  'articleEditor.save': 'Save Draft',
  'articleEditor.titlePlaceholder': 'Title',
  'articleEditor.slugPlaceholder': 'Slug',
  'articleEditor.coverPlaceholder': 'Cover URL',
  'articleEditor.chooseCategory': 'Choose category...',
  'articleEditor.tags': 'Tags:',
  'articleEditor.summary': 'Summary',
  'articleEditor.content': 'Body (Markdown)...',
  'articleEditor.depsError': 'Failed to load categories and tags',
  'articleEditor.saveError': 'Failed to save',

  'taxonomy.listCategoryError': 'Failed to load category list',
  'taxonomy.listTagError': 'Failed to load tag list',
  'taxonomy.submitError': 'Operation failed',
  'taxonomy.deleteCategoryError': 'Delete failed; related articles may exist',
  'taxonomy.deleteTagError': 'Delete failed',
  'taxonomy.confirmDelete': 'Confirm deletion?',
  'taxonomy.add': 'Add',

  'users.account': 'Account',
  'users.nickname': 'Nickname',
  'users.role': 'Role',
  'users.activate': 'Enable',
  'users.disable': 'Disable',
  'users.confirmStatus': 'Confirm to {action} this user?',
  'users.loadError': 'Failed to load user list',
  'users.actionError': 'Operation failed',

  'logs.user': 'User',
  'logs.event': 'Event',
  'logs.source': 'Source',
  'logs.loadError': 'Failed to load logs',
};

const dictionaries: Record<Language, Record<TranslationKey, string>> = {
  zh,
  en,
};

export const translate = (language: Language, key: TranslationKey) => {
  return dictionaries[language][key];
};

export const formatText = (template: string, values: Record<string, string | number>) => {
  return Object.entries(values).reduce(
    (message, [key, value]) => message.split(`{${key}}`).join(String(value)),
    template,
  );
};

export const getDateLocale = (language: Language) => (language === 'zh' ? 'zh-CN' : 'en-US');

export const getArticleStatusLabel = (language: Language, status: string) => {
  const labels: Record<string, Record<Language, string>> = {
    draft: { zh: '草稿', en: 'Draft' },
    published: { zh: '已发布', en: 'Published' },
    archived: { zh: '归档', en: 'Archived' },
  };

  return labels[status]?.[language] ?? status;
};

export const getUserRoleLabel = (language: Language, role: string) => {
  const labels: Record<string, Record<Language, string>> = {
    admin: { zh: '掌卷', en: 'Admin' },
    user: { zh: '墨客', en: 'Writer' },
  };

  return labels[role]?.[language] ?? role;
};

export const getUserStatusLabel = (language: Language, status: string) => {
  const labels: Record<string, Record<Language, string>> = {
    active: { zh: '活跃', en: 'Active' },
    disabled: { zh: '流放', en: 'Disabled' },
  };

  return labels[status]?.[language] ?? status;
};
