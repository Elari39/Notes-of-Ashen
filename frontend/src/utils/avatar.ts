export const normalizeAvatarUrl = (value?: string | null) => value?.trim() ?? '';

export const isHttpAvatarUrl = (value?: string | null) => {
  const url = normalizeAvatarUrl(value);
  if (!url) return false;

  try {
    const parsed = new URL(url);
    return parsed.protocol === 'http:' || parsed.protocol === 'https:';
  } catch {
    return false;
  }
};
