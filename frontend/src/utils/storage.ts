type StorageAdapter = Pick<Storage, 'getItem' | 'setItem' | 'removeItem'>;

type SafeStorage = StorageAdapter & {
  removeByPrefix: (prefix: string) => void;
};

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
    removeByPrefix: (prefix) => {
      const keys = new Set<string>();
      for (const key of memory.keys()) {
        if (key.startsWith(prefix)) {
          keys.add(key);
        }
      }
      try {
        const storage = resolveStorage() as (StorageAdapter & { length?: number; key?: (index: number) => string | null }) | undefined;
        if (storage && typeof storage.length === 'number' && typeof storage.key === 'function') {
          for (let index = 0; index < storage.length; index += 1) {
            const key = storage.key(index);
            if (key?.startsWith(prefix)) {
              keys.add(key);
            }
          }
        }
        keys.forEach((key) => storage?.removeItem(key));
      } catch {
        unavailable = true;
      }
      keys.forEach((key) => memory.delete(key));
    },
  };
};

export const safeLocalStorage = createSafeStorage(() => (
  typeof window === 'undefined' ? undefined : window.localStorage
));

export const safeSessionStorage = createSafeStorage(() => (
  typeof window === 'undefined' ? undefined : window.sessionStorage
));
