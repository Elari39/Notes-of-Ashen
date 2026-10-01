import { createClient } from 'redis';
import { e2eEnv } from './env';

const withRedis = async <T>(operation: (client: ReturnType<typeof createClient>) => Promise<T>): Promise<T> => {
  const client = createClient({ url: e2eEnv.redisUrl });
  // node-redis 要求显式 error listener；具体调用仍会将连接/命令错误抛给测试。
  client.on('error', () => undefined);
  try {
    await client.connect();
    return await operation(client);
  } finally {
    if (client.isOpen) {
      await client.quit();
    }
  }
};

const readCaptcha = async (purpose: 'login' | 'register', captchaID: string): Promise<string> => {
  const normalizedID = captchaID.trim();
  if (!normalizedID) {
    throw new Error(`Cannot read an empty ${purpose} captcha ID from test Redis.`);
  }
  const key = `captcha:${purpose}:${normalizedID}`;
  const code = await withRedis((client) => client.get(key));
  if (!code) {
    throw new Error(`${purpose} captcha ${normalizedID} was not found in test Redis.`);
  }
  return code;
};

export const readLoginCaptcha = (captchaID: string): Promise<string> => readCaptcha('login', captchaID);

export const readRegisterCaptcha = (captchaID: string): Promise<string> => readCaptcha('register', captchaID);

// 多个串行场景共享同一 IP。开始两次登录的场景前，等待前序场景占用的
// 真实 5 次/分钟窗口释放额度；不删除计数、不重试登录、不更改限流配置。
export const waitForLoginQuota = async (requests: number): Promise<void> => {
  const waitMs = await withRedis(async (client) => {
    let wait = 0;
    for (const key of await client.keys('rate_limit:auth_login:*')) {
      const count = Number(await client.get(key));
      if (count + requests > 5) {
        const ttl = await client.pTTL(key);
        if (ttl === -1 || ttl > 60_000) throw new Error('Unexpected login rate-limit window');
        wait = Math.max(wait, ttl);
      }
    }
    return wait;
  });
  if (waitMs > 0) await new Promise((resolve) => setTimeout(resolve, waitMs + 50));
};

// 隔离环境的真实服务端失效注入；不改签名、接口响应或限流配置。
export const expireIssuedAccessTokens = async (): Promise<number> => {
  const cutoff = Math.floor(Date.now() / 1000);
  await withRedis((client) => client.set('auth:access:not-before', String(cutoff), { EX: 120 }));
  return cutoff;
};

export const seedRegisterEmailCode = async (email: string, code: string): Promise<void> => {
  const normalizedEmail = email.trim().toLowerCase();
  if (!normalizedEmail || !code.trim()) {
    throw new Error('Registration email and verification code are required for Redis test setup.');
  }
  await withRedis(async (client) => {
    await client.set(`verify_code:register:${normalizedEmail}`, code.trim(), { EX: 300 });
  });
};
