import React from 'react';

type Size = 'sm' | 'md';

type CommonProps = {
  size?: Size;
  invalid?: boolean;
  prefix?: React.ReactNode;
  suffix?: React.ReactNode;
  fieldClassName?: string;
};

type InputProps = CommonProps &
  Omit<React.InputHTMLAttributes<HTMLInputElement>, 'size' | 'prefix'> & {
    asTextarea?: false;
  };

type TextareaProps = CommonProps &
  Omit<React.TextareaHTMLAttributes<HTMLTextAreaElement>, 'size' | 'prefix'> & {
    asTextarea: true;
    rows?: number;
  };

export type TextFieldProps = InputProps | TextareaProps;

const sizeInputClass: Record<Size, string> = {
  sm: 'min-h-[2.75rem] px-3 py-2 text-sm',
  md: 'min-h-[2.75rem] px-3.5 py-2.5 text-sm',
};

const wrapperBase =
  'flex items-center gap-2 rounded-md border bg-paper transition-[border-color,box-shadow] duration-fast ease-paper focus-within:ring-[3px] focus-within:ring-[var(--inline-code-bg)]';

/**
 * 统一表单输入。
 * - 下划线风格（沿用站点账号链路），focus 内置 ochre + outline
 * - invalid 时下划线 + 文字变 danger
 * - prefix / suffix slot 接受图标 / 后缀按钮
 * - asTextarea 渲染 textarea，去掉下划线只保留边框
 */
const TextField = React.forwardRef<HTMLInputElement | HTMLTextAreaElement, TextFieldProps>(
  (props, ref) => {
    const {
      size = 'md',
      invalid = false,
      prefix,
      suffix,
      fieldClassName = '',
      className = '',
      ...rest
    } = props;

    const toneClass = invalid
      ? 'border-ember text-ember placeholder:text-ember/60'
      : 'border-hairline text-ink hover:border-muted focus-within:border-ochre';

    if (props.asTextarea) {
      const { asTextarea: _asTextarea, rows = 4, ...textareaRest } = rest as React.TextareaHTMLAttributes<HTMLTextAreaElement> & { asTextarea: true; rows?: number };
      void _asTextarea;
      return (
        <textarea
          ref={ref as React.Ref<HTMLTextAreaElement>}
          rows={rows}
          aria-invalid={invalid || undefined}
          className={[
            'block w-full resize-y rounded-md border bg-paper px-3.5 py-3 text-sm leading-relaxed transition-[border-color,box-shadow] duration-fast ease-paper',
            'focus:outline-hidden focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-ochre',
            invalid
              ? 'border-ember text-ember placeholder:text-ember/60'
              : 'border-hairline text-ink placeholder:text-muted hover:border-muted focus:border-ochre focus:ring-[3px] focus:ring-[var(--inline-code-bg)]',
            fieldClassName,
            className,
          ]
            .filter(Boolean)
            .join(' ')}
          {...textareaRest}
        />
      );
    }

    const inputRest = rest as React.InputHTMLAttributes<HTMLInputElement>;

    return (
      <div
        className={[wrapperBase, toneClass, fieldClassName].filter(Boolean).join(' ')}
      >
        {prefix && <span className="text-ink-light opacity-70">{prefix}</span>}
        <input
          ref={ref as React.Ref<HTMLInputElement>}
          aria-invalid={invalid || undefined}
          className={[
            'block w-full bg-transparent placeholder:text-ink-light placeholder:opacity-50 focus:outline-hidden',
            sizeInputClass[size],
            invalid ? 'placeholder:text-ember/60' : '',
            className,
          ]
            .filter(Boolean)
            .join(' ')}
          {...inputRest}
        />
        {suffix && <span className="text-ink-light opacity-70">{suffix}</span>}
      </div>
    );
  },
);

TextField.displayName = 'TextField';

export default TextField;
