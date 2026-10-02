import assert from 'node:assert/strict';
import { once } from 'node:events';
import { createServer, request, type RequestListener, type Server } from 'node:http';
import { connect } from 'node:net';
import test from 'node:test';
import { createNetworkCutProxy } from '../e2e/helpers/networkCutProxy.ts';

const listen = async (handler: RequestListener) => {
  const server = createServer(handler);
  server.listen(0, '127.0.0.1');
  await once(server, 'listening');
  const address = server.address();
  assert.ok(address && typeof address !== 'string');
  return { server, origin: `http://127.0.0.1:${address.port}` };
};

const closeServer = (server: Server) => new Promise<void>((resolve, reject) => {
  server.close((error) => error ? reject(error) : resolve());
  server.closeAllConnections();
});

const send = (origin: string, path = '/', body?: string) => new Promise<{
  status: number | undefined;
  body: string;
  contentType: string | undefined;
}>((resolve, reject) => {
  const client = request(new URL(origin), {
    path,
    method: body === undefined ? 'GET' : 'POST',
    headers: { 'Content-Type': 'text/plain' },
    agent: false,
  }, (response) => {
    let text = '';
    response.setEncoding('utf8');
    response.on('data', (chunk: string) => { text += chunk; });
    response.on('error', reject);
    response.on('end', () => resolve({ status: response.statusCode, body: text, contentType: response.headers['content-type'] }));
  });
  client.on('error', reject);
  client.end(body);
});

test('代理保留请求方法、路径、正文与真实源站响应', { timeout: 5000 }, async (t) => {
  const upstream = await listen((request, response) => {
    let body = '';
    request.setEncoding('utf8');
    request.on('data', (chunk: string) => { body += chunk; });
    request.on('end', () => {
      response.writeHead(410, { 'Content-Type': 'application/json' });
      response.end(JSON.stringify({ method: request.method, path: request.url, body }));
    });
  });
  t.after(() => closeServer(upstream.server));
  const proxy = await createNetworkCutProxy(upstream.origin);
  t.after(() => proxy.close());
  const response = await send(proxy.origin, '/api/v1/articles/7?view=reader', '正文');
  assert.equal(response.status, 410);
  assert.equal(response.contentType, 'application/json');
  assert.deepEqual(JSON.parse(response.body), { method: 'POST', path: '/api/v1/articles/7?view=reader', body: '正文' });
});

test('断开代理终止在途响应及上游连接，不合成 HTTP 错误响应', { timeout: 5000 }, async (t) => {
  const upstream = await listen((_request, response) => {
    response.writeHead(200, { 'Content-Type': 'text/plain' });
    response.write('unfinished');
  });
  t.after(() => closeServer(upstream.server));
  const proxy = await createNetworkCutProxy(upstream.origin);
  t.after(() => proxy.close());
  const received = once(upstream.server, 'request');
  const interrupted = assert.rejects(send(proxy.origin));
  const [upstreamRequest] = await received;
  // TCP reset 是断连的预期结果；等待 close，而不是让 events.once 因 error 提前拒绝。
  const upstreamClosed = new Promise<void>((resolve) => upstreamRequest.socket.once('close', resolve));
  proxy.disconnect();
  await Promise.all([interrupted, upstreamClosed]);
});

test('断开代理关闭空闲连接并阻止新请求到达源站', { timeout: 5000 }, async (t) => {
  let requests = 0;
  const upstream = await listen((_request, response) => {
    requests++;
    response.end('online');
  });
  t.after(() => closeServer(upstream.server));
  const proxy = await createNetworkCutProxy(upstream.origin);
  t.after(() => proxy.close());
  assert.equal((await send(proxy.origin)).body, 'online');
  const socket = connect({ host: '127.0.0.1', port: Number(new URL(proxy.origin).port) });
  t.after(() => socket.destroy());
  await once(socket, 'connect');
  const idleClosed = once(socket, 'close');
  proxy.disconnect();
  proxy.disconnect();
  await idleClosed;
  await assert.rejects(send(proxy.origin, '/healthz?uncached=1'));
  assert.equal(requests, 1);
});

test('代理拒绝客户端提供的绝对 URL，关闭可重复且释放监听端口', { timeout: 5000 }, async (t) => {
  let requests = 0;
  const upstream = await listen((_request, response) => { requests++; response.end('online'); });
  t.after(() => closeServer(upstream.server));
  const proxy = await createNetworkCutProxy(upstream.origin);
  t.after(() => proxy.close());
  await assert.rejects(send(proxy.origin, `${upstream.origin}/unexpected`));
  assert.equal(requests, 0);
  await Promise.all([proxy.close(), proxy.close()]);
  await proxy.close();
  await assert.rejects(send(proxy.origin));
});
