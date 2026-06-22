import React from 'react';
import { motion, AnimatePresence, useReducedMotion } from 'framer-motion';
import Button from './Button';

export type SaveBarProps = {
  /** 是否显示 SaveBar（通常等同于 dirty 状态） */
  open: boolean;
  /** 是否正在保存 */
  saving?: boolean;
  /** 主操作按钮文字 */
  saveLabel: string;
  /** 取消/丢弃按钮文字 */
  cancelLabel?: string;
  /** 状态文案（如 "尚未保存" / "已保存 3 分钟前"） */
  status?: React.ReactNode;
  onSave: () => void;
  onCancel?: () => void;
};

/**
 * 粘底保存提示：放在编辑器底部，dirty 时滑入。
 * 配合 useSubmit 的 saving 状态显示 loading button。
 */
const SaveBar: React.FC<SaveBarProps> = ({
  open,
  saving = false,
  saveLabel,
  cancelLabel,
  status,
  onSave,
  onCancel,
}) => {
  const reduce = useReducedMotion();
  const motionProps = reduce
    ? { initial: { opacity: 1 }, animate: { opacity: 1 }, exit: { opacity: 0 } }
    : { initial: { y: 24, opacity: 0 }, animate: { y: 0, opacity: 1 }, exit: { y: 24, opacity: 0 } };

  return (
    <AnimatePresence>
      {open && (
        <motion.div
          {...motionProps}
          transition={{ duration: 0.24, ease: 'easeOut' }}
          className="pointer-events-none fixed inset-x-0 bottom-4 z-[80] flex justify-center px-4"
        >
          <div className="pointer-events-auto flex items-center gap-4 border border-mountain-grey bg-paper px-4 py-3 shadow-md">
            {status && (
              <span className="text-xs tracking-widest text-ink-light">{status}</span>
            )}
            <div className="flex items-center gap-2">
              {onCancel && (
                <Button variant="ghost" size="sm" onClick={onCancel} disabled={saving}>
                  {cancelLabel}
                </Button>
              )}
              <Button variant="primary" size="sm" onClick={onSave} loading={saving}>
                {saveLabel}
              </Button>
            </div>
          </div>
        </motion.div>
      )}
    </AnimatePresence>
  );
};

export default SaveBar;
