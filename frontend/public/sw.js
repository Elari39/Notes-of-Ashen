const CACHE_VERSION = 'notes-of-ashen-v1';
const SHELL_CACHE = `${CACHE_VERSION}:shell`;
const ARTICLE_CACHE = `${CACHE_VERSION}:articles`;
const SHELL_ASSETS = ['/', '/index.html', '/favicon.svg', '/favicon.png', '/pwa-192.png', '/pwa-512.png', '/manifest.webmanifest'];
const ARTICLE_DETAIL_RE = /^\/api\/v1\/articles\/\d+$/;

self.addEventListener('install', (event) => {
  event.waitUntil(
    caches.open(SHELL_CACHE)
      .then((cache) => cache.addAll(SHELL_ASSETS))
      .then(() => self.skipWaiting()),
  );
});

self.addEventListener('activate', (event) => {
  event.waitUntil(
    caches.keys()
      .then((keys) => Promise.all(keys
        .filter((key) => !key.startsWith(CACHE_VERSION))
        .map((key) => caches.delete(key))))
      .then(() => self.clients.claim()),
  );
});

// 文章删除/状态变更后，前端通过 postMessage 通知清空详情缓存，避免 offline 命中陈旧内容。
self.addEventListener('message', (event) => {
  if (event.data && event.data.type === 'CLEAR_ARTICLE_CACHE') {
    event.waitUntil(caches.delete(ARTICLE_CACHE));
  }
});

self.addEventListener('fetch', (event) => {
  const request = event.request;
  if (request.method !== 'GET') {
    return;
  }

  const url = new URL(request.url);
  if (url.origin !== self.location.origin) {
    return;
  }
  if (url.pathname.startsWith('/api/v1/admin')
    || url.pathname.startsWith('/api/v1/auth')
    || url.pathname.startsWith('/api/v1/users')) {
    return;
  }

  if (ARTICLE_DETAIL_RE.test(url.pathname)) {
    event.respondWith(networkFirst(request, ARTICLE_CACHE));
    return;
  }

  if (request.mode === 'navigate') {
    event.respondWith(navigationFallback(request));
    return;
  }

  if (isStaticAsset(url.pathname)) {
    event.respondWith(cacheFirst(request, SHELL_CACHE));
  }
});

const cacheFirst = async (request, cacheName) => {
  const cached = await caches.match(request);
  if (cached) {
    return cached;
  }
  const response = await fetch(request);
  if (response.ok) {
    const cache = await caches.open(cacheName);
    await cache.put(request, response.clone());
  }
  return response;
};

const networkFirst = async (request, cacheName) => {
  const cache = await caches.open(cacheName);
  try {
    const response = await fetch(request);
    if (response.ok) {
      await cache.put(request, response.clone());
      await trimCache(cacheName, 24);
    }
    return response;
  } catch {
    const cached = await cache.match(request);
    if (cached) {
      return cached;
    }
    throw new Error('offline and no cached response');
  }
};

const navigationFallback = async (request) => {
  try {
    const response = await fetch(request);
    const cache = await caches.open(SHELL_CACHE);
    if (response.ok) {
      cache.put('/index.html', response.clone());
    }
    return response;
  } catch {
    const cached = await caches.match('/index.html');
    return cached || new Response('Offline', {
      status: 503,
      statusText: 'Offline',
      headers: { 'Content-Type': 'text/plain; charset=utf-8' },
    });
  }
};

const trimCache = async (cacheName, maxItems) => {
  const cache = await caches.open(cacheName);
  const keys = await cache.keys();
  if (keys.length <= maxItems) {
    return;
  }
  await cache.delete(keys[0]);
  return trimCache(cacheName, maxItems);
};

const isStaticAsset = (pathname) => /\.(?:js|css|svg|png|jpg|jpeg|webp|woff2?)$/i.test(pathname);
