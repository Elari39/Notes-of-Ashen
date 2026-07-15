import type { PaginatedResp } from '../types';

type PaginatedResponse<T> = {
  data: PaginatedResp<T>;
};

type PageLoader<T> = (
  params: { page: number; size: number },
  signal?: AbortSignal,
) => Promise<PaginatedResponse<T>>;

type CollectPaginatedOptions = {
  pageSize?: number;
  signal?: AbortSignal;
};

const abortError = () => {
  const error = new Error('The operation was aborted');
  error.name = 'AbortError';
  return error;
};

const throwIfAborted = (signal?: AbortSignal) => {
  if (signal?.aborted) {
    throw abortError();
  }
};

export const collectPaginated = async <T>(
  loadPage: PageLoader<T>,
  options: CollectPaginatedOptions = {},
): Promise<T[]> => {
  const pageSize = Math.max(1, Math.trunc(options.pageSize || 100));
  const items: T[] = [];
  let page = 1;
  let total = Number.POSITIVE_INFINITY;

  while (items.length < total) {
    throwIfAborted(options.signal);
    const response = await loadPage({ page, size: pageSize }, options.signal);
    throwIfAborted(options.signal);

    const pageItems = response.data.items || [];
    total = response.data.total;
    items.push(...pageItems);
    if (pageItems.length === 0) {
      break;
    }
    page += 1;
  }
  return items;
};
