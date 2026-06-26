import React from 'react';

type Variant = 'primary' | 'ghost' | 'subtle' | 'danger' | 'link';
type Size = 'sm' | 'md' | 'lg';

type ButtonBaseProps = {
  variant?: Variant;
  size?: Size;
  loading?: boolean;
  iconBefore?: React.ReactNode;
  iconAfter?: React.ReactNode;
  fullWidth?: boolean;
  className?: string;
};

type ButtonAsButton = ButtonBaseProps &
  Omit<React.ButtonHTMLAttributes<HTMLButtonElement>, keyof ButtonBaseProps | 'type'> & {
    as?: 'button';
    type?: 'button' | 'submit' | 'reset';
  };

type ButtonAsAnchor = ButtonBaseProps &
  Omit<React.AnchorHTMLAttributes<HTMLAnchorElement>, keyof ButtonBaseProps> & {
    as: 'a';
  };

export type ButtonProps = ButtonAsButton | ButtonAsAnchor;

const variantClass: Record<Variant, string> = {
  // 实心主按钮：ochre 底 + paper 字
  primary:
    'bg-ochre text-paper border border-ochre hover:brightness-95 active:brightness-90 disabled:opacity-50 disabled:cursor-not-allowed disabled:hover:brightness-100',
  // 边框次按钮：项目最常见的形态
  ghost:
    'bg-transparent text-ink border border-mountain-grey hover:border-ochre hover:text-ochre active:bg-[var(--paper-soft)] disabled:opacity-50 disabled:cursor-not-allowed disabled:hover:border-mountain-grey disabled:hover:text-ink',
  // 无边框淡按钮：辅助 / 工具栏
  subtle:
    'bg-transparent text-ink-light border border-transparent hover:bg-[var(--paper-soft)] hover:text-ink active:bg-[var(--paper-muted)] disabled:opacity-50 disabled:cursor-not-allowed disabled:hover:bg-transparent',
  // 危险操作：ghost 形态 + danger 文字 / hover 反白
  danger:
    'bg-transparent text-ember border border-[var(--ember-soft)] hover:bg-ember hover:text-paper hover:border-ember active:brightness-95 disabled:opacity-50 disabled:cursor-not-allowed disabled:hover:bg-transparent disabled:hover:text-ember',
  // 纯文字按钮，配合 article a 链接样式
  link:
    'bg-transparent text-ochre border-0 underline decoration-[var(--code-border)] underline-offset-4 hover:decoration-ochre disabled:opacity-50 disabled:cursor-not-allowed',
};

const sizeClass: Record<Size, string> = {
  sm: 'min-h-[2rem] px-3 py-1 text-xs tracking-widest',
  md: 'min-h-[2.5rem] px-4 py-1.5 text-sm tracking-widest',
  lg: 'min-h-[3rem] px-6 py-2 text-base tracking-widest',
};

const baseClass =
  'inline-flex items-center justify-center gap-2 transition-colors duration-fast ease-paper focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-ochre';

const Spinner: React.FC<{ size: Size }> = ({ size }) => {
  const dim = size === 'sm' ? 12 : size === 'lg' ? 16 : 14;
  return (
    <span
      aria-hidden="true"
      className="inline-block animate-spin rounded-full border-2 border-current border-r-transparent"
      style={{ width: dim, height: dim }}
    />
  );
};

/**
 * 统一按钮组件。
 * - variant: primary / ghost(默认) / subtle / danger / link
 * - size: sm / md(默认) / lg
 * - loading: 内置 spinner + disabled，文案不动以避免宽度跳变
 * - 全部走 CSS 变量，自动暗色适配
 */
const Button = React.forwardRef<HTMLElement, ButtonProps>((props, ref) => {
  const {
    variant = 'ghost',
    size = 'md',
    loading = false,
    iconBefore,
    iconAfter,
    fullWidth = false,
    className = '',
    children,
    ...rest
  } = props;

  const composed = [
    baseClass,
    variantClass[variant],
    sizeClass[size],
    fullWidth ? 'w-full' : '',
    className,
  ]
    .filter(Boolean)
    .join(' ');

  const content = (
    <>
      {loading ? <Spinner size={size} /> : iconBefore}
      <span className={loading ? 'opacity-80' : ''}>{children}</span>
      {!loading && iconAfter}
    </>
  );

  if (props.as === 'a') {
    const { as: _as, ...anchorRest } = rest as React.AnchorHTMLAttributes<HTMLAnchorElement> & { as?: 'a' };
    void _as;
    return (
      <a
        ref={ref as React.Ref<HTMLAnchorElement>}
        className={composed}
        aria-busy={loading || undefined}
        {...anchorRest}
      >
        {content}
      </a>
    );
  }

  const { type = 'button', disabled, ...buttonRest } = rest as React.ButtonHTMLAttributes<HTMLButtonElement>;
  return (
    <button
      ref={ref as React.Ref<HTMLButtonElement>}
      type={type}
      disabled={disabled || loading}
      aria-busy={loading || undefined}
      className={composed}
      {...buttonRest}
    >
      {content}
    </button>
  );
});

Button.displayName = 'Button';

export default Button;
