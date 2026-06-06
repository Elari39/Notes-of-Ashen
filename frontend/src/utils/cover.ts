export const normalizeCoverUrl = (coverUrl?: string) => {
  const trimmed = coverUrl?.trim() || '';

  if (!trimmed) {
    return '';
  }

  if (/^https?:\/\//i.test(trimmed) || trimmed.startsWith('/')) {
    return trimmed;
  }

  return `/${trimmed}`;
};
