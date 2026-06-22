import React from 'react';
import * as SwitchPrimitive from '@radix-ui/react-switch';

type Size = 'sm' | 'md';

export type SwitchProps = {
  checked: boolean;
  onCheckedChange: (next: boolean) => void;
  label?: string;
  size?: Size;
  disabled?: boolean;
  id?: string;
  className?: string;
  'aria-describedby'?: string;
};

const sizeRail: Record<Size, string> = {
  sm: 'h-4 w-7',
  md: 'h-5 w-9',
};

const sizeThumb: Record<Size, string> = {
  sm: 'h-3 w-3 data-[state=checked]:translate-x-3 data-[state=unchecked]:translate-x-0',
  md: 'h-4 w-4 data-[state=checked]:translate-x-4 data-[state=unchecked]:translate-x-0',
};

/**
 * Switch（基于 @radix-ui/react-switch）。
 * Radix 自动处理 role=switch / aria-checked / 键盘 Space。
 */
const Switch: React.FC<SwitchProps> = ({
  checked,
  onCheckedChange,
  label,
  size = 'md',
  disabled = false,
  id,
  className = '',
  'aria-describedby': ariaDescribedBy,
}) => {
  return (
    <SwitchPrimitive.Root
      id={id}
      checked={checked}
      onCheckedChange={onCheckedChange}
      disabled={disabled}
      aria-label={label}
      aria-describedby={ariaDescribedBy}
      className={[
        'relative inline-flex shrink-0 items-center border transition-colors duration-fast ease-paper',
        'focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-ochre',
        sizeRail[size],
        'data-[state=checked]:border-ochre data-[state=checked]:bg-ochre',
        'data-[state=unchecked]:border-mountain-grey data-[state=unchecked]:bg-[var(--paper-soft)] data-[state=unchecked]:hover:border-ink-light',
        disabled ? 'opacity-50 cursor-not-allowed' : 'cursor-pointer',
        className,
      ]
        .filter(Boolean)
        .join(' ')}
    >
      <SwitchPrimitive.Thumb
        className={[
          'pointer-events-none ml-0.5 inline-block transform bg-paper transition-transform duration-fast ease-paper',
          sizeThumb[size],
        ].join(' ')}
      />
    </SwitchPrimitive.Root>
  );
};

export default Switch;
