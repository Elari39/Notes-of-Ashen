// 前端持久化 visitor id，用于点赞等场景的后端去重 hash 输入，防止换 UA 刷赞。
// 存于 localStorage，换设备/清缓存会重新生成（与原 IP+UA 方案属同等权衡）。

const VISITOR_ID_KEY = 'notesOfAshen.visitorId';

function genFallback(): string {
  // crypto.randomUUID 在 https/localhost 下可用；兜底用时间戳+随机数。
  const rand = Math.random().toString(36).slice(2);
  const time = Date.now().toString(36);
  return `${time}-${rand}`;
}

export function getVisitorId(): string {
  try {
    let id = localStorage.getItem(VISITOR_ID_KEY);
    if (!id) {
      id = typeof crypto !== 'undefined' && typeof crypto.randomUUID === 'function'
        ? crypto.randomUUID()
        : genFallback();
      localStorage.setItem(VISITOR_ID_KEY, id);
    }
    return id;
  } catch {
    // localStorage 不可用（隐私模式等）时返回空串，后端回退到 IP+UA hash。
    return '';
  }
}
