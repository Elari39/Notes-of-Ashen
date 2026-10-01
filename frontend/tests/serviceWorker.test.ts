import assert from 'node:assert/strict';
import { readFileSync } from 'node:fs';
import test from 'node:test';
import vm from 'node:vm';

// 执行真实 sw.js 的 fetch handler；这是离线协议单测，浏览器验证另在 E2E 执行。
const createWorker = () => {
  const entries = new Map<string, Response>();
  const handlers = new Map<string, (event: { request: Request; respondWith: (response: Promise<Response>) => void }) => void>();
  let fetchResult: Response | Error = new Response('published');
  const cache = {
    match: async (request: Request) => entries.get(request.url)?.clone(),
    put: async (request: Request, response: Response) => { entries.set(request.url, response); },
    keys: async () => [...entries.keys()].map((url) => new Request(url)),
    delete: async (request: Request, options?: { ignoreSearch: boolean }) => {
      for (const key of entries.keys()) {
        if (key === request.url || (options?.ignoreSearch && new URL(key).pathname === new URL(request.url).pathname)) entries.delete(key);
      }
      return true;
    },
  };
  const context = vm.createContext({
    URL, Response,
    self: { location: { origin: 'https://blog.example' }, addEventListener: (name: string, handler: typeof handlers extends Map<string, infer V> ? V : never) => handlers.set(name, handler) },
    caches: { open: async () => cache },
    fetch: async () => {
      if (fetchResult instanceof Error) throw fetchResult;
      return fetchResult.clone();
    },
  });
  vm.runInContext(readFileSync(new URL('../public/sw.js', import.meta.url), 'utf8'), context);
  return {
    setResult: (result: Response | Error) => { fetchResult = result; },
    intercepts: (path: string) => {
      let intercepted = false;
      handlers.get('fetch')!({ request: new Request(`https://blog.example${path}`), respondWith: (promise) => { intercepted = true; void promise.catch(() => undefined); } });
      return intercepted;
    },
    request: (path: string) => new Promise<Response>((resolve, reject) => {
      handlers.get('fetch')!({ request: new Request(`https://blog.example${path}`), respondWith: (promise) => { promise.then(resolve, reject); } });
    }),
  };
};

test('媒体绕过 Service Worker 缓存，遵从源站撤回策略', () => {
  const worker = createWorker();
  assert.equal(worker.intercepts('/media/picture.png'), false);
  assert.equal(worker.intercepts('/media/picture.png?version=1'), false);
});

for (const status of [401, 403, 404, 410]) {
  test(`收到 ${status} 后离线不能恢复旧文章及查询参数变体`, async () => {
    const worker = createWorker();
    assert.equal(await (await worker.request('/api/v1/articles/7')).text(), 'published');
    await worker.request('/api/v1/articles/7?view=reader');
    worker.setResult(new Response('withdrawn', { status }));
    assert.equal((await worker.request('/api/v1/articles/7')).status, status);
    worker.setResult(new Error('offline'));
    await assert.rejects(worker.request('/api/v1/articles/7'), /offline and no cached response/);
    await assert.rejects(worker.request('/api/v1/articles/7?view=reader'), /offline and no cached response/);
  });
}

test('网络故障保留离线阅读能力；5xx 不删除旧缓存或冒充 200', async () => {
  const worker = createWorker();
  await worker.request('/api/v1/articles/7');
  worker.setResult(new Response('unavailable', { status: 503 }));
  assert.equal((await worker.request('/api/v1/articles/7')).status, 503);
  worker.setResult(new Error('offline'));
  assert.equal(await (await worker.request('/api/v1/articles/7')).text(), 'published');
});
