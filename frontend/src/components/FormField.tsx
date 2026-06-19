import React from 'react';

type FormFieldProps = {
  id: string;
  label: string;
  error?: string;
  hint?: string;
  className?: string;
  children: React.ReactElement;
};

/**
 * 表单字段容器：label + 受控 input + 字段级错误文案。
 * 通过 cloneElement 注入 id / aria-invalid / aria-describedby，保持调用方受控逻辑不变。
 */
const FormField: React.FC<FormFieldProps> = ({ id, label, error, hint, className = '', children }) => {
  const describedBy = error ? `${id}-error` : hint ? `${id}-hint` : undefined;

  return (
    <div className={className}>
      <label htmlFor={id} className="mb-2 block text-xs tracking-widest text-ink-light">
        {label}
      </label>
      {React.cloneElement(children, {
        id,
        'aria-invalid': Boolean(error) || undefined,
        'aria-describedby': describedBy,
      })}
      {hint && !error && (
        <p id={`${id}-hint`} className="mt-2 text-xs leading-relaxed text-ink-light opacity-70">
          {hint}
        </p>
      )}
      {error && (
        <p id={`${id}-error`} role="alert" className="mt-2 border-l-2 border-ochre bg-[var(--notice-bg)] px-3 py-2 text-xs text-ochre">
          {error}
        </p>
      )}
    </div>
  );
};

export default FormField;
