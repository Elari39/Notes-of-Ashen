import React from 'react';
import * as Dialog from '@radix-ui/react-dialog';

type Size = 'sm' | 'md' | 'lg';

export type ModalProps = {
  open: boolean;
  onOpenChange: (next: boolean) => void;
  title: React.ReactNode;
  description?: React.ReactNode;
  size?: Size;
  hideCloseButton?: boolean;
  /** 是否点击 overlay 关闭，默认 true */
  closeOnOverlayClick?: boolean;
  footer?: React.ReactNode;
  children?: React.ReactNode;
  closeLabel?: string;
};

const sizeClass: Record<Size, string> = {
  sm: 'max-w-sm',
  md: 'max-w-md',
  lg: 'max-w-2xl',
};

/**
 * 通用模态框（基于 @radix-ui/react-dialog）。
 * - focus trap / ESC / aria-modal / portal 都由 Radix 提供
 * - 视觉走站点 paper + ink + mountain-grey 风格，零圆角到 radius-md
 */
const Modal: React.FC<ModalProps> = ({
  open,
  onOpenChange,
  title,
  description,
  size = 'md',
  hideCloseButton = false,
  closeOnOverlayClick = true,
  footer,
  children,
  closeLabel = 'Close',
}) => {
  return (
    <Dialog.Root open={open} onOpenChange={onOpenChange}>
      <Dialog.Portal>
        <Dialog.Overlay
          className="fixed inset-0 z-[120] bg-[var(--paper-muted)] backdrop-blur-sm data-[state=open]:animate-in data-[state=open]:fade-in motion-reduce:animate-none"
          onClick={(e) => {
            if (!closeOnOverlayClick) e.preventDefault();
          }}
        />
        <Dialog.Content
          onInteractOutside={(e) => {
            if (!closeOnOverlayClick) e.preventDefault();
          }}
          className={[
            'fixed left-1/2 top-1/2 z-[121] w-[calc(100vw-2rem)] -translate-x-1/2 -translate-y-1/2',
            sizeClass[size],
            'overflow-hidden rounded-xl border border-hairline bg-paper shadow-lg',
            'focus:outline-none',
          ].join(' ')}
        >
          <div className="flex items-start justify-between gap-4 border-b border-mountain-grey px-5 py-3.5">
            <div className="flex-1 min-w-0">
              <Dialog.Title className="font-display text-2xl leading-tight text-ink">
                {title}
              </Dialog.Title>
              {description && (
                <Dialog.Description className="mt-1 text-xs leading-relaxed text-ink-light">
                  {description}
                </Dialog.Description>
              )}
            </div>
            {!hideCloseButton && (
              <Dialog.Close
                aria-label={closeLabel}
                className="-mr-1 inline-flex h-11 w-11 shrink-0 items-center justify-center rounded-full text-ink-light transition-colors hover:bg-surface-soft hover:text-ink focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-ochre"
              >
                ✕
              </Dialog.Close>
            )}
          </div>
          {children && (
            <div className="px-5 py-4 text-sm leading-relaxed text-ink-light">{children}</div>
          )}
          {footer && (
            <div className="flex flex-wrap items-center justify-end gap-2 border-t border-mountain-grey px-5 py-3">
              {footer}
            </div>
          )}
        </Dialog.Content>
      </Dialog.Portal>
    </Dialog.Root>
  );
};

export default Modal;
