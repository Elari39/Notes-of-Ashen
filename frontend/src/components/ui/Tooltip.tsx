import React from 'react';
import * as TooltipPrimitive from '@radix-ui/react-tooltip';

export type TooltipProps = {
  content: React.ReactNode;
  children: React.ReactNode;
  side?: 'top' | 'bottom' | 'left' | 'right';
  delayDuration?: number;
  /** 触摸设备隐藏（hover: none） */
  disableOnTouch?: boolean;
  className?: string;
};

/**
 * 提示气泡。需要在 App 根部挂 <TooltipPrimitive.Provider />。
 */
const Tooltip: React.FC<TooltipProps> = ({
  content,
  children,
  side = 'top',
  delayDuration = 300,
  disableOnTouch = true,
  className = '',
}) => {
  return (
    <TooltipPrimitive.Root delayDuration={delayDuration} disableHoverableContent={disableOnTouch}>
      <TooltipPrimitive.Trigger asChild>{children}</TooltipPrimitive.Trigger>
      <TooltipPrimitive.Portal>
        <TooltipPrimitive.Content
          side={side}
          sideOffset={6}
          className={[
            'z-[130] border border-mountain-grey bg-paper px-2.5 py-1 text-xs tracking-widest text-ink shadow-sm',
            'data-[state=delayed-open]:animate-in data-[state=closed]:animate-out',
            'data-[state=delayed-open]:fade-in data-[state=closed]:fade-out',
            'motion-reduce:animate-none',
            className,
          ]
            .filter(Boolean)
            .join(' ')}
        >
          {content}
        </TooltipPrimitive.Content>
      </TooltipPrimitive.Portal>
    </TooltipPrimitive.Root>
  );
};

export const TooltipProvider = TooltipPrimitive.Provider;

export default Tooltip;
