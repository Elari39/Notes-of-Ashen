import { createServer, request as httpRequest, type ClientRequest } from 'node:http';
import { request as httpsRequest } from 'node:https';
import type { Socket } from 'node:net';

/** 仅用于 E2E 的固定源站代理；断开 TCP 连接，让真实 Service Worker 处理网络失败。 */
export const createNetworkCutProxy = async (targetURL: string) => {
  const target = new URL(targetURL);
  if (!['http:', 'https:'].includes(target.protocol)) {
    throw new Error('Network cut proxy requires an HTTP(S) origin.');
  }
  const forwardRequest = target.protocol === 'https:' ? httpsRequest : httpRequest;
  const clients = new Set<Socket>();
  const upstreams = new Set<ClientRequest>();
  let disconnected = false;
  let closing: Promise<void> | undefined;

  const server = createServer((request, response) => {
    // 只接受本站路径，客户端不能通过 absolute-form URL 更换转发目标。
    if (disconnected || !request.url?.startsWith('/') || request.url.startsWith('//')) {
      request.socket.destroy();
      return;
    }
    const upstream = forwardRequest({
      hostname: target.hostname,
      port: target.port || (target.protocol === 'https:' ? 443 : 80),
      method: request.method,
      path: request.url,
      headers: request.headers,
      agent: false,
    }, (upstreamResponse) => {
      upstreamResponse.on('error', () => response.destroy());
      response.writeHead(upstreamResponse.statusCode!, upstreamResponse.headers);
      upstreamResponse.pipe(response);
    });
    upstreams.add(upstream);
    upstream.once('close', () => upstreams.delete(upstream));
    upstream.on('error', () => response.destroy());
    request.on('error', () => upstream.destroy());
    response.once('close', () => upstream.destroy());
    request.pipe(upstream);
  });
  server.on('connection', (socket) => {
    clients.add(socket);
    socket.once('close', () => clients.delete(socket));
    if (disconnected) socket.destroy();
  });

  await new Promise<void>((resolve, reject) => {
    server.once('error', reject);
    server.listen(0, '127.0.0.1', () => {
      server.removeListener('error', reject);
      resolve();
    });
  });
  const address = server.address();
  if (!address || typeof address === 'string') {
    server.close();
    throw new Error('Network cut proxy did not bind a TCP port.');
  }

  const disconnect = () => {
    disconnected = true;
    for (const upstream of upstreams) upstream.destroy();
    for (const socket of clients) socket.destroy();
  };

  return {
    origin: `http://127.0.0.1:${address.port}`,
    disconnect,
    close: () => {
      if (!closing) {
        disconnect();
        closing = new Promise<void>((resolve, reject) => {
          server.close((error) => error ? reject(error) : resolve());
        });
      }
      return closing;
    },
  };
};
