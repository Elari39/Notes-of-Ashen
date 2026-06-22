import React from 'react';
import { create } from 'zustand';
import type { ConfirmTone } from '../components/ui/ConfirmDialog';

export type ConfirmOptions = {
  title: string;
  description?: React.ReactNode;
  confirmLabel?: string;
  cancelLabel?: string;
  tone?: ConfirmTone;
  closeLabel?: string;
};

type PendingItem = ConfirmOptions & {
  id: number;
  resolve: (ok: boolean) => void;
};

type ConfirmState = {
  current: PendingItem | null;
  enqueue: (options: ConfirmOptions) => Promise<boolean>;
  resolveCurrent: (ok: boolean) => void;
};

let nextId = 0;

export const useConfirmStore = create<ConfirmState>((set, get) => ({
  current: null,
  enqueue: (options) =>
    new Promise<boolean>((resolve) => {
      nextId += 1;
      const item: PendingItem = { ...options, id: nextId, resolve };
      const { current } = get();
      // 同时只支持一个；新请求顶替旧请求并以 false 结算旧请求
      if (current) current.resolve(false);
      set({ current: item });
    }),
  resolveCurrent: (ok) => {
    const { current } = get();
    if (!current) return;
    current.resolve(ok);
    set({ current: null });
  },
}));

/**
 * 命令式确认 hook。
 * 使用：const confirm = useConfirm(); const ok = await confirm({ title, tone: 'danger' });
 */
export const useConfirm = () => {
  return React.useCallback((options: ConfirmOptions) => {
    return useConfirmStore.getState().enqueue(options);
  }, []);
};
