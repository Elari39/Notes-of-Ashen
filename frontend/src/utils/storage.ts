type StorageAdapter = Pick<Storage, 'getItem' | 'setItem' | 'removeItem'>;

type SafeStorage = StorageAdapter;

export const createSafeStorage = (resolveStorage: () => StorageAdapter | undefined): SafeStorage => {
  const memory = new Map<string, string>();
  let unavailable = false;

  return {
    getItem: (key) => {
      if (unavailable) {
        return memory.get(key) ?? null;
      }
      try {
        const storage = resolveStorage();
        if (!storage) {
          return memory.get(key) ?? null;
        }
        const value = storage.getItem(key);
        if (value === null) {
          memory.delete(key);
        } else {
          memory.set(key, value);
        }
        return value;
      } catch {
        unavailable = true;
        return memory.get(key) ?? null;
      }
    },
    setItem: (key, value) => {
      const normalized = String(value);
      memory.set(key, normalized);
      if (unavailable) {
        return;
      }
      try {
        resolveStorage()?.setItem(key, normalized);
      } catch {
        unavailable = true;
      }
    },
    removeItem: (key) => {
      memory.delete(key);
      if (unavailable) {
        return;
      }
      try {
        resolveStorage()?.removeItem(key);
      } catch {
        unavailable = true;
      }
    },
  };
};

export const safeLocalStorage = createSafeStorage(() => (
  typeof window === 'undefined' ? undefined : window.localStorage
));

export const safeSessionStorage = createSafeStorage(() => (
  typeof window === 'undefined' ? undefined : window.sessionStorage
));
