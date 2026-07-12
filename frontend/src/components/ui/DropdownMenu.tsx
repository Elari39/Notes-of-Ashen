import React from 'react';
import * as DropdownMenuPrimitive from '@radix-ui/react-dropdown-menu';

export type DropdownMenuItem = {
  key: string;
  label: React.ReactNode;
  onSelect: () => void;
  /** danger 项：文字变 danger，hover 反白 */
  tone?: 'default' | 'danger';
  disabled?: boolean;
  /** 在该项之前插入一条分隔线 */
  separatorBefore?: boolean;
};

export type DropdownMenuProps = {
  trigger: React.ReactNode;
  items: DropdownMenuItem[];
  /** 对齐方式 */
  align?: 'start' | 'center' | 'end';
  side?: 'top' | 'bottom' | 'left' | 'right';
};

/**
 * 下拉菜单（基于 @radix-ui/react-dropdown-menu）。
 * - 用于 admin 行内"更多操作"，避免按钮长串
 */
const DropdownMenu: React.FC<DropdownMenuProps> = ({
  trigger,
  items,
  align = 'end',
  side = 'bottom',
}) => {
  return (
    <DropdownMenuPrimitive.Root>
      <DropdownMenuPrimitive.Trigger asChild>{trigger}</DropdownMenuPrimitive.Trigger>
      <DropdownMenuPrimitive.Portal>
        <DropdownMenuPrimitive.Content
          align={align}
          side={side}
          sideOffset={4}
          className="z-[125] min-w-[11rem] rounded-lg border border-hairline bg-paper p-1.5 shadow-md focus:outline-none data-[state=open]:animate-in data-[state=open]:fade-in motion-reduce:animate-none"
        >
          {items.map((item, idx) => (
            <React.Fragment key={item.key}>
              {item.separatorBefore && idx !== 0 && (
                <DropdownMenuPrimitive.Separator className="my-1 h-px bg-mountain-grey" />
              )}
              <DropdownMenuPrimitive.Item
                disabled={item.disabled}
                onSelect={(e) => {
                  e.preventDefault();
                  item.onSelect();
                }}
                className={[
                  'block rounded-sm px-3 py-2.5 text-xs font-medium cursor-pointer transition-colors duration-fast',
                  'data-[highlighted]:bg-surface-soft data-[highlighted]:outline-none',
                  'data-[disabled]:opacity-50 data-[disabled]:cursor-not-allowed',
                  item.tone === 'danger'
                    ? 'text-ember data-[highlighted]:bg-[var(--ember-soft)] data-[highlighted]:text-ember'
                    : 'text-ink data-[highlighted]:text-ochre',
                ].join(' ')}
              >
                {item.label}
              </DropdownMenuPrimitive.Item>
            </React.Fragment>
          ))}
        </DropdownMenuPrimitive.Content>
      </DropdownMenuPrimitive.Portal>
    </DropdownMenuPrimitive.Root>
  );
};

export default DropdownMenu;
