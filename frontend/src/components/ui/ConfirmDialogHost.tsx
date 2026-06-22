import React from 'react';
import ConfirmDialog from './ConfirmDialog';
import { useConfirmStore } from '../../hooks/useConfirm';

/**
 * 全局确认对话框宿主，挂在 App 根部一次。
 */
const ConfirmDialogHost: React.FC = () => {
  const current = useConfirmStore((state) => state.current);
  const resolveCurrent = useConfirmStore((state) => state.resolveCurrent);

  return (
    <ConfirmDialog
      open={Boolean(current)}
      onOpenChange={(next) => {
        if (!next) resolveCurrent(false);
      }}
      title={current?.title ?? ''}
      description={current?.description}
      confirmLabel={current?.confirmLabel}
      cancelLabel={current?.cancelLabel}
      tone={current?.tone}
      closeLabel={current?.closeLabel}
      onConfirm={() => resolveCurrent(true)}
    />
  );
};

export default ConfirmDialogHost;
