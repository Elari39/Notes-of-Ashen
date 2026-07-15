import type { ReactNode } from 'react';

type SettingsCardProps = {
  title: ReactNode;
  description?: ReactNode;
  action?: ReactNode;
  children?: ReactNode;
  className?: string;
  contentClassName?: string;
};

const SettingsCard = ({
  title,
  description,
  action,
  children,
  className = '',
  contentClassName = '',
}: SettingsCardProps) => (
  <section className={`border border-mountain-grey bg-[var(--paper-soft)] p-5 ${className}`.trim()}>
    <div className={`flex flex-col gap-4 ${action ? 'md:flex-row md:items-center md:justify-between' : ''}`.trim()}>
      <div>
        <h4 className="text-base font-bold tracking-widest text-ink">{title}</h4>
        {description && (
          <p className="mt-2 text-sm leading-relaxed text-ink-light opacity-80">{description}</p>
        )}
      </div>
      {action && <div className="shrink-0">{action}</div>}
    </div>
    {children && <div className={`mt-5 ${contentClassName}`.trim()}>{children}</div>}
  </section>
);

export const SettingsActions = ({
  children,
  className = '',
}: {
  children: ReactNode;
  className?: string;
}) => (
  <div className={`flex flex-wrap items-center gap-3 ${className}`.trim()}>
    {children}
  </div>
);

export default SettingsCard;
