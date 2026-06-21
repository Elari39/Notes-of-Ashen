export const generateSlug = (value: string) => {
  const slug = value
    .normalize('NFKD')
    .replace(/[̀-ͯ]/g, '')
    .toLowerCase()
    .replace(/[^a-z0-9]+/g, '-')
    .replace(/^-+|-+$/g, '')
    .replace(/-{2,}/g, '-');
  // 纯中文/非拉丁标题经上面处理后可能为空串，回退 untitled-{时间戳}，
  // 避免空 slug 导致 URL 冲突或后端 slug required 报错（P4-23）。
  return slug || `untitled-${Date.now()}`;
};
