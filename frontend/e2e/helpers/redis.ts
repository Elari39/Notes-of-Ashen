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

export const seedRegisterEmailCode = async (email: string, code: string): Promise<void> => {
  const normalizedEmail = email.trim().toLowerCase();
  if (!normalizedEmail || !code.trim()) {
    throw new Error('Registration email and verification code are required for Redis test setup.');
  }
  await withRedis(async (client) => {
    await client.set(`verify_code:register:${normalizedEmail}`, code.trim(), { EX: 300 });
  });
};
