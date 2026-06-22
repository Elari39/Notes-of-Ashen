import React from 'react';

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
  sm: 'h-3 w-3',
  md: 'h-4 w-4',
};

const sizeThumbOn: Record<Size, string> = {
  sm: 'translate-x-3',
  md: 'translate-x-4',
};

/**
 * 自研 Switch（阶段 2 临时实现，阶段 3 用 @radix-ui/react-switch 替换内部实现，对外 API 保持兼容）。
 * - role=switch + aria-checked
 * - 走 CSS 变量，自动暗色
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
    <button
      type="button"
      id={id}
      role="switch"
      aria-checked={checked}
      aria-label={label}
      aria-describedby={ariaDescribedBy}
      disabled={disabled}
      onClick={() => !disabled && onCheckedChange(!checked)}
      className={[
        'relative inline-flex shrink-0 items-center border transition-colors duration-fast ease-paper',
        'focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-ochre',
        sizeRail[size],
        checked
          ? 'border-ochre bg-ochre'
          : 'border-mountain-grey bg-[var(--paper-soft)] hover:border-ink-light',
        disabled ? 'opacity-50 cursor-not-allowed' : 'cursor-pointer',
        className,
      ]
        .filter(Boolean)
        .join(' ')}
    >
      <span
        aria-hidden="true"
        className={[
          'absolute left-0.5 inline-block transform bg-paper transition-transform duration-fast ease-paper',
          sizeThumb[size],
          checked ? sizeThumbOn[size] : 'translate-x-0',
        ].join(' ')}
      />
    </button>
  );
};

export default Switch;
