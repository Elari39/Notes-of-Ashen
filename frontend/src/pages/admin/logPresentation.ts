import { getArticleStatusLabel, translate, type TranslationKey } from '../../i18n.ts';
import type { Language } from '../../store/preferences.ts';
import type { Log } from '../../types/index.ts';

export type LogTone = 'neutral' | 'ochre' | 'success' | 'warning' | 'danger' | 'info';

type EventPresentation = {
  labelKey: TranslationKey;
  tone: LogTone;
};

const eventPresentations: Record<string, EventPresentation> = {
  'article.created': { labelKey: 'logs.event.articleCreated', tone: 'success' },
  'article.updated': { labelKey: 'logs.event.articleUpdated', tone: 'ochre' },
  'article.deleted': { labelKey: 'logs.event.articleDeleted', tone: 'danger' },
  'article.status_updated': { labelKey: 'logs.event.articleStatusUpdated', tone: 'warning' },
  'article.version_restored': { labelKey: 'logs.event.articleVersionRestored', tone: 'ochre' },
  'auth.verify_code_sent': { labelKey: 'logs.event.authVerifyCodeSent', tone: 'info' },
  'user.verify_code_sent': { labelKey: 'logs.event.userVerifyCodeSent', tone: 'info' },
  'user.registered': { labelKey: 'logs.event.userRegistered', tone: 'success' },
  'user.logged_in': { labelKey: 'logs.event.userLoggedIn', tone: 'info' },
  'user.logged_out': { labelKey: 'logs.event.userLoggedOut', tone: 'neutral' },
  'user.password_reset': { labelKey: 'logs.event.userPasswordReset', tone: 'warning' },
  'token.mismatch_logout': { labelKey: 'logs.event.tokenMismatchLogout', tone: 'danger' },
};

export const LOG_EVENT_TYPES = Object.keys(eventPresentations);

export const getLogEventPresentation = (eventType: string, language: Language) => {
  const presentation = eventPresentations[eventType];
  if (!presentation) {
    return {
      label: eventType || translate(language, 'logs.unknownEvent'),
      tone: 'neutral' as LogTone,
    };
  }
  return {
    label: translate(language, presentation.labelKey),
    tone: presentation.tone,
  };
};

const resourceLabelKeys: Record<string, TranslationKey> = {
  article: 'logs.resource.article',
  user: 'logs.resource.user',
  email: 'logs.resource.email',
};

export const formatLogResource = (log: Pick<Log, 'resourceType' | 'resourceId'>, language: Language) => {
  const resourceType = log.resourceType.trim();
  const labelKey = resourceLabelKeys[resourceType];
  const label = labelKey
    ? translate(language, labelKey)
    : resourceType || translate(language, 'logs.unknownResource');
  return log.resourceId ? `${label} #${log.resourceId}` : label;
};

export type LogMetadataEntry = {
  key: string;
  value: string;
};

export type ParsedLogMetadata = {
  entries: LogMetadataEntry[];
  raw: string;
  invalid: boolean;
};

export const parseLogMetadata = (metadata?: string): ParsedLogMetadata => {
  const raw = metadata?.trim() ?? '';
  if (!raw) {
    return { entries: [], raw: '', invalid: false };
  }
  try {
    const parsed: unknown = JSON.parse(raw);
    if (!parsed || typeof parsed !== 'object' || Array.isArray(parsed)) {
      return { entries: [], raw, invalid: true };
    }
    const entries = Object.entries(parsed).map(([key, value]) => ({
      key,
      value: typeof value === 'string' ? value : JSON.stringify(value),
    }));
    return { entries, raw, invalid: false };
  } catch {
    return { entries: [], raw, invalid: true };
  }
};

const metadataLabelKeys: Record<string, TranslationKey> = {
  purpose: 'logs.metadata.purpose',
  status: 'logs.metadata.status',
  versionNo: 'logs.metadata.versionNo',
};

export const getLogMetadataLabel = (key: string, language: Language) => {
  const labelKey = metadataLabelKeys[key];
  return labelKey ? translate(language, labelKey) : key;
};

const purposeLabels: Record<string, Record<Language, string>> = {
  login: { zh: '登录', en: 'Login' },
  register: { zh: '注册', en: 'Registration' },
  reset_password: { zh: '重置密码', en: 'Password reset' },
  change_password: { zh: '修改密码', en: 'Password change' },
  update_email: { zh: '修改邮箱', en: 'Email update' },
};

export const formatLogMetadataValue = (key: string, value: string, language: Language) => {
  if (key === 'status') {
    return getArticleStatusLabel(language, value);
  }
  if (key === 'purpose') {
    return purposeLabels[value]?.[language] ?? value;
  }
  return value;
};

const matchVersion = (userAgent: string, expression: RegExp, name: string) => {
  const matched = userAgent.match(expression);
  return matched ? `${name} ${matched[1]}` : '';
};

export const getClientSummary = (userAgent: string, language: Language) => {
  const ua = userAgent.trim();
  if (!ua) {
    return translate(language, 'logs.unknownClient');
  }
  const browser =
    matchVersion(ua, /Edg\/([\d.]+)/, 'Edge') ||
    matchVersion(ua, /(?:Chrome|CriOS)\/([\d.]+)/, 'Chrome') ||
    matchVersion(ua, /(?:Firefox|FxiOS)\/([\d.]+)/, 'Firefox') ||
    (ua.includes('Safari/') ? matchVersion(ua, /Version\/([\d.]+)/, 'Safari') : '') ||
    matchVersion(ua, /curl\/([\d.]+)/, 'curl') ||
    translate(language, 'logs.unknownBrowser');

  let system = translate(language, 'logs.unknownSystem');
  if (/Windows NT/.test(ua)) system = 'Windows';
  else if (/Android/.test(ua)) system = 'Android';
  else if (/iPhone|iPad|iPod/.test(ua)) system = 'iOS';
  else if (/Mac OS X/.test(ua)) system = 'macOS';
  else if (/Linux/.test(ua)) system = 'Linux';
  return `${browser} · ${system}`;
};

export const dateToUTCBoundary = (value: string, endExclusive: boolean) => {
  if (!value) return undefined;
  const parts = value.split('-').map(Number);
  if (parts.length !== 3 || parts.some((part) => !Number.isInteger(part))) return undefined;
  const [year, month, day] = parts;
  const date = new Date(year, month - 1, day);
  if (
    Number.isNaN(date.getTime()) ||
    date.getFullYear() !== year ||
    date.getMonth() !== month - 1 ||
    date.getDate() !== day
  ) {
    return undefined;
  }
  if (endExclusive) date.setDate(date.getDate() + 1);
  return date.toISOString();
};
