import React from 'react';
import Modal from './Modal';
import Button from './Button';

export type ConfirmTone = 'default' | 'danger';

export type ConfirmDialogProps = {
  open: boolean;
  onOpenChange: (next: boolean) => void;
  title: string;
  description?: React.ReactNode;
  confirmLabel?: string;
  cancelLabel?: string;
  tone?: ConfirmTone;
  onConfirm: () => void | Promise<void>;
  closeLabel?: string;
};

/**
 * 确认对话框：替换原生 window.confirm。
 * - tone='danger' 时确认按钮变红
 * - onConfirm 返回 Promise 时自动管理 loading；reject 时不关闭
 */
const ConfirmDialog: React.FC<ConfirmDialogProps> = ({
  open,
  onOpenChange,
  title,
  description,
  confirmLabel = 'OK',
  cancelLabel = 'Cancel',
  tone = 'default',
  onConfirm,
  closeLabel,
}) => {
  const [loading, setLoading] = React.useState(false);

  const handleConfirm = async () => {
    try {
      setLoading(true);
      await Promise.resolve(onConfirm());
      onOpenChange(false);
    } catch {
      // 错误向上抛由调用方处理（toast / InlineNotice），不静默关闭
    } finally {
      setLoading(false);
    }
  };

  return (
    <Modal
      open={open}
      onOpenChange={(next) => {
        if (loading) return;
        onOpenChange(next);
      }}
      title={title}
      description={description}
      size="sm"
      closeLabel={closeLabel}
      closeOnOverlayClick={!loading}
      footer={
        <>
          <Button
            variant="subtle"
            size="sm"
            onClick={() => onOpenChange(false)}
            disabled={loading}
          >
            {cancelLabel}
          </Button>
          <Button
            variant={tone === 'danger' ? 'danger' : 'primary'}
            size="sm"
            loading={loading}
            onClick={handleConfirm}
          >
            {confirmLabel}
          </Button>
        </>
      }
    />
  );
};

export default ConfirmDialog;
