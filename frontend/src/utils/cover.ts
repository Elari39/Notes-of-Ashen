export const normalizeCoverUrl = (coverUrl?: string) => {
  const trimmed = coverUrl?.trim() || '';

  if (!trimmed) {
    return '';
  }

  return isHttpCoverUrl(trimmed) ? trimmed : '';
};

export const isValidCoverUrl = (coverUrl?: string) => {
  const trimmed = coverUrl?.trim() || '';
  return trimmed === '' || isHttpCoverUrl(trimmed);
};

const isHttpCoverUrl = (value: string) => {
  try {
    const url = new URL(value);
    return url.protocol === 'http:' || url.protocol === 'https:';
  } catch {
    return false;
  }
};
