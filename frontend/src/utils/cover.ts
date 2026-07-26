export const normalizeCoverUrl = (coverUrl?: string) => {
  const trimmed = coverUrl?.trim() || '';

  if (!trimmed) {
    return '';
  }

  return isHttpCoverUrl(trimmed) || isLocalMediaUrl(trimmed) ? trimmed : '';
};

export const isValidCoverUrl = (coverUrl?: string) => {
  const trimmed = coverUrl?.trim() || '';
  return trimmed === '' || isHttpCoverUrl(trimmed) || isLocalMediaUrl(trimmed);
};

const isLocalMediaUrl = (value: string) => /^\/media\/[a-f0-9]{64}\.(jpg|png|gif|webp|avif)$/.test(value);

const isHttpCoverUrl = (value: string) => {
  try {
    const url = new URL(value);
    return url.protocol === 'http:' || url.protocol === 'https:';
  } catch {
    return false;
  }
};
