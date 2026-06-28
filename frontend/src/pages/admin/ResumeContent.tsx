import React, { useEffect, useMemo, useRef, useState } from 'react';
import InlineNotice from '../../components/InlineNotice';
import MarkdownRenderer from '../../components/MarkdownRenderer';
import PagePendingState from '../../components/RoutePending';
import { getAdminResumePage, updateAdminResumePage } from '../../api/siteSettings';
import { usePreferenceStore } from '../../store/preferences';
import type { ResumeEducation, ResumeExperience, ResumePage, ResumeSkill } from '../../types';
import { getErrorMessage } from '../../utils/error';
import { useSubmit } from '../../hooks/useSubmit';
import { useDebouncedValue } from '../../hooks/useDebouncedValue';
import { formatText, translate } from '../../i18n';

const emptyResume: ResumePage = {
  title: '',
  subtitle: '',
  contentMarkdown: '',
  experiences: [],
  educations: [],
  skills: [],
};

const AdminResumeContent: React.FC = () => {
  const language = usePreferenceStore((state) => state.language);
  const t = (key: Parameters<typeof translate>[1]) => translate(language, key);
  // 用 ref 持有最新 language，避免切换语言时重新拉取简历内容（数据与语言无关）。
  const languageRef = useRef(language);
  languageRef.current = language;
  const [saved, setSaved] = useState<ResumePage | null>(null);
  const [draft, setDraft] = useState<ResumePage>(emptyResume);
  const [isLoading, setIsLoading] = useState(true);
  const [loadError, setLoadError] = useState('');
  // 预览防抖：每次按键只更新 draft，预览用延迟值，避免全文重解析。
  const debouncedPreviewMarkdown = useDebouncedValue(draft.contentMarkdown, 250);

  useEffect(() => {
    let active = true;
    setIsLoading(true);
    setLoadError('');
    getAdminResumePage()
      .then((res) => {
        if (!active) {
          return;
        }
        const next = normalizeResumePage(res.data, translate(languageRef.current, 'resumeAdmin.defaultTitle'));
        setSaved(next);
        setDraft(next);
      })
      .catch((e: unknown) => {
        if (active) {
          setLoadError(getErrorMessage(e, translate(languageRef.current, 'resumeAdmin.loadError')));
        }
      })
      .finally(() => {
        if (active) {
          setIsLoading(false);
        }
      });
    return () => {
      active = false;
    };
  }, []);

  const hasChanges = useMemo(
    () => Boolean(saved) && JSON.stringify(saved) !== JSON.stringify(draft),
    [draft, saved],
  );

  const {
    submit: handleSave,
    submitting: isSaving,
    error: saveError,
    reset: resetSaveError,
  } = useSubmit({
    handler: async () => {
      const payload = normalizeResumeForSave(draft);
      const res = await updateAdminResumePage(payload);
      return normalizeResumePage(res.data, t('resumeAdmin.defaultTitle'));
    },
    successMessage: t('resumeAdmin.saved'),
    errorFallback: t('resumeAdmin.saveError'),
    onSuccess: (next) => {
      setSaved(next);
      setDraft(next);
    },
  });

  const error = loadError || saveError;

  const handleReset = () => {
    if (saved) {
      setDraft(saved);
    }
    setLoadError('');
    resetSaveError();
  };

  return (
    <div>
      <div className="mb-8 border-b border-mountain-grey pb-4">
        <h3 className="text-2xl font-bold tracking-widest text-ink">{t('resumeAdmin.title')}</h3>
      </div>

      <InlineNotice message={error} className="mb-6" />

      {isLoading && (
        <PagePendingState
          variant={saved ? 'inline' : 'admin'}
          label={t('resumeAdmin.loading')}
        />
      )}
      {(!isLoading || saved) && (
        <div className="space-y-8">
          <div className="grid grid-cols-1 gap-5 md:grid-cols-2">
            <TextInput label={t('resumeAdmin.pageTitle')} value={draft.title} disabled={isSaving} onChange={(title) => setDraft((prev) => ({ ...prev, title }))} />
            <TextInput label={t('resumeAdmin.subtitle')} value={draft.subtitle} disabled={isSaving} onChange={(subtitle) => setDraft((prev) => ({ ...prev, subtitle }))} />
          </div>

          <div className="grid min-h-[30rem] grid-cols-1 gap-6 lg:grid-cols-2">
            <section className="flex min-h-[24rem] flex-col border border-mountain-grey p-4">
              <div className="mb-3 text-sm font-bold tracking-widest text-ink">{t('resumeAdmin.editor')}</div>
              <textarea
                value={draft.contentMarkdown}
                onChange={(event) => setDraft((prev) => ({ ...prev, contentMarkdown: event.target.value }))}
                disabled={isSaving}
                placeholder={t('resumeAdmin.placeholder')}
                className="min-h-0 flex-1 resize-none bg-transparent text-ink-light outline-none disabled:cursor-not-allowed disabled:opacity-50"
              />
            </section>
            <section className="min-h-[24rem] overflow-y-auto border border-mountain-grey bg-[var(--paper-soft)] p-4">
              <div className="mb-3 text-sm font-bold tracking-widest text-ink">{t('resumeAdmin.preview')}</div>
  {debouncedPreviewMarkdown.trim() ? (
    <MarkdownRenderer content={debouncedPreviewMarkdown} />
  ) : (
    <p className="py-12 text-center text-sm tracking-[0.2em] text-ink-light">{t('resumeAdmin.emptyPreview')}</p>
  )}
            </section>
          </div>

          <ResumeArraySection
            title={t('resumeAdmin.experiences')}
            addLabel={t('resumeAdmin.addExperience')}
            emptyLabel={t('resumeAdmin.emptyExperiences')}
            disabled={isSaving}
            items={draft.experiences}
            createItem={createEmptyExperience}
            onChange={(experiences) => setDraft((prev) => ({ ...prev, experiences }))}
            itemKey={resumeItemKey('experience')}
            renderItem={(item, index, collapsed, onToggleCollapsed, onChange, onRemove) => (
              <ExperienceEditor item={item} index={index} disabled={isSaving} collapsed={collapsed} onToggleCollapsed={onToggleCollapsed} onChange={onChange} onRemove={onRemove} />
            )}
          />

          <ResumeArraySection
            title={t('resumeAdmin.educations')}
            addLabel={t('resumeAdmin.addEducation')}
            emptyLabel={t('resumeAdmin.emptyEducations')}
            disabled={isSaving}
            items={draft.educations}
            createItem={createEmptyEducation}
            onChange={(educations) => setDraft((prev) => ({ ...prev, educations }))}
            itemKey={resumeItemKey('education')}
            renderItem={(item, index, collapsed, onToggleCollapsed, onChange, onRemove) => (
              <EducationEditor item={item} index={index} disabled={isSaving} collapsed={collapsed} onToggleCollapsed={onToggleCollapsed} onChange={onChange} onRemove={onRemove} />
            )}
          />

          <ResumeArraySection
            title={t('resumeAdmin.skills')}
            addLabel={t('resumeAdmin.addSkill')}
            emptyLabel={t('resumeAdmin.emptySkills')}
            disabled={isSaving}
            items={draft.skills}
            createItem={createEmptySkill}
            onChange={(skills) => setDraft((prev) => ({ ...prev, skills }))}
            itemKey={resumeItemKey('skill')}
            renderItem={(item, index, collapsed, onToggleCollapsed, onChange, onRemove) => (
              <SkillEditor item={item} index={index} disabled={isSaving} collapsed={collapsed} onToggleCollapsed={onToggleCollapsed} onChange={onChange} onRemove={onRemove} />
            )}
          />

          <div className="flex flex-wrap gap-3">
            <button
              type="button"
              onClick={() => { void handleSave(); }}
              disabled={isSaving || !hasChanges}
              className="border border-ink px-4 py-2 text-sm tracking-widest text-ink transition-colors hover:bg-ink hover:text-paper disabled:cursor-not-allowed disabled:opacity-50"
            >
              {isSaving ? t('resumeAdmin.saving') : t('resumeAdmin.save')}
            </button>
            <button
              type="button"
              onClick={handleReset}
              disabled={isSaving || !hasChanges}
              className="border border-mountain-grey px-4 py-2 text-sm tracking-widest text-ink-light transition-colors hover:border-ochre hover:text-ochre disabled:cursor-not-allowed disabled:opacity-50"
            >
              {t('resumeAdmin.reset')}
            </button>
          </div>
        </div>
      )}
    </div>
  );
};

export default AdminResumeContent;

type ResumeArraySectionProps<T> = {
  title: string;
  addLabel: string;
  emptyLabel: string;
  disabled: boolean;
  items: T[];
  createItem: () => T;
  onChange: (items: T[]) => void;
  itemKey: (item: T, index: number) => string;
  renderItem: (
    item: T,
    index: number,
    collapsed: boolean,
    onToggleCollapsed: () => void,
    onChange: (patch: Partial<T>) => void,
    onRemove: () => void,
  ) => React.ReactNode;
};

const ResumeArraySection = <T,>({
  title,
  addLabel,
  emptyLabel,
  disabled,
  items,
  createItem,
  onChange,
  itemKey,
  renderItem,
}: ResumeArraySectionProps<T>) => {
  const language = usePreferenceStore((state) => state.language);
  const t = (key: Parameters<typeof translate>[1]) => translate(language, key);
  const [collapsedKeys, setCollapsedKeys] = useState<Set<string>>(() => new Set());

  const handleAdd = () => {
    onChange([...items, createItem()]);
  };

  const handleRemove = (index: number, key: string) => {
    onChange(items.filter((_, currentIndex) => currentIndex !== index));
    setCollapsedKeys((prev) => {
      const next = new Set(prev);
      next.delete(key);
      return next;
    });
  };

  const toggleCollapsed = (key: string) => {
    setCollapsedKeys((prev) => {
      const next = new Set(prev);
      if (next.has(key)) {
        next.delete(key);
      } else {
        next.add(key);
      }
      return next;
    });
  };

  return (
    <section className="border border-mountain-grey bg-[var(--paper-soft)] p-4 md:p-5">
      <div className="mb-5 flex flex-col gap-3 md:flex-row md:items-center md:justify-between">
        <h4 className="text-base font-bold tracking-widest text-ink">{title}</h4>
        <button
          type="button"
          onClick={handleAdd}
          disabled={disabled}
          className="border border-ochre px-3 py-1.5 text-sm tracking-widest text-ochre transition-colors hover:bg-ochre hover:text-paper disabled:cursor-not-allowed disabled:opacity-50"
        >
          {addLabel}
        </button>
      </div>
      {items.length === 0 ? (
        <p className="text-sm tracking-[0.18em] text-ink-light">{emptyLabel}</p>
      ) : (
        <div className="space-y-4">
          {items.map((item, index) => {
            const key = itemKey(item, index);
            const collapsed = collapsedKeys.has(key);
            return (
              <React.Fragment key={key}>
                {renderItem(
                  item,
                  index,
                  collapsed,
                  () => toggleCollapsed(key),
                  (patch) => onChange(items.map((current, currentIndex) => currentIndex === index ? { ...current, ...patch } : current)),
                  () => handleRemove(index, key),
                )}
              </React.Fragment>
            );
          })}
        </div>
      )}
      {items.length > 0 && (
        <p className="mt-4 text-xs leading-6 tracking-[0.14em] text-ink-light opacity-70">
          {t('resumeAdmin.collapseHint')}
        </p>
      )}
    </section>
  );
};

const ExperienceEditor: React.FC<{
  item: ResumeExperience;
  index: number;
  disabled: boolean;
  collapsed: boolean;
  onToggleCollapsed: () => void;
  onChange: (patch: Partial<ResumeExperience>) => void;
  onRemove: () => void;
}> = ({ item, index, disabled, collapsed, onToggleCollapsed, onChange, onRemove }) => {
  const language = usePreferenceStore((state) => state.language);
  const t = (key: Parameters<typeof translate>[1]) => translate(language, key);
  return (
  <article className="border border-mountain-grey bg-paper p-4">
    <EditorHeader title={item.role || formatText(t('resumeAdmin.itemNo'), { value: index + 1 })} collapsed={collapsed} disabled={disabled} onToggleCollapsed={onToggleCollapsed} onRemove={onRemove} />
    {!collapsed && (
      <div className="grid grid-cols-1 gap-4 md:grid-cols-2">
        <TextInput label={t('resumeAdmin.role')} value={item.role} disabled={disabled} onChange={(role) => onChange({ role })} />
        <TextInput label={t('resumeAdmin.organization')} value={item.organization} disabled={disabled} onChange={(organization) => onChange({ organization })} />
        <TextInput label={t('resumeAdmin.location')} value={item.location} disabled={disabled} onChange={(location) => onChange({ location })} />
        <DateRangeInputs start={item.startDate} end={item.endDate} disabled={disabled} onChange={onChange} />
        <TextArea label={t('resumeAdmin.description')} value={item.description} disabled={disabled} onChange={(description) => onChange({ description })} className="md:col-span-2" />
        <TextArea label={t('resumeAdmin.highlights')} value={item.highlights.join('\n')} disabled={disabled} onChange={(value) => onChange({ highlights: splitLines(value) })} className="md:col-span-2" />
      </div>
    )}
  </article>
  );
};

const EducationEditor: React.FC<{
  item: ResumeEducation;
  index: number;
  disabled: boolean;
  collapsed: boolean;
  onToggleCollapsed: () => void;
  onChange: (patch: Partial<ResumeEducation>) => void;
  onRemove: () => void;
}> = ({ item, index, disabled, collapsed, onToggleCollapsed, onChange, onRemove }) => {
  const language = usePreferenceStore((state) => state.language);
  const t = (key: Parameters<typeof translate>[1]) => translate(language, key);
  return (
  <article className="border border-mountain-grey bg-paper p-4">
    <EditorHeader title={item.school || formatText(t('resumeAdmin.itemNo'), { value: index + 1 })} collapsed={collapsed} disabled={disabled} onToggleCollapsed={onToggleCollapsed} onRemove={onRemove} />
    {!collapsed && (
      <div className="grid grid-cols-1 gap-4 md:grid-cols-2">
        <TextInput label={t('resumeAdmin.school')} value={item.school} disabled={disabled} onChange={(school) => onChange({ school })} />
        <TextInput label={t('resumeAdmin.degree')} value={item.degree} disabled={disabled} onChange={(degree) => onChange({ degree })} />
        <TextInput label={t('resumeAdmin.major')} value={item.major} disabled={disabled} onChange={(major) => onChange({ major })} />
        <TextInput label={t('resumeAdmin.location')} value={item.location} disabled={disabled} onChange={(location) => onChange({ location })} />
        <DateRangeInputs start={item.startDate} end={item.endDate} disabled={disabled} onChange={onChange} />
        <TextArea label={t('resumeAdmin.description')} value={item.description} disabled={disabled} onChange={(description) => onChange({ description })} className="md:col-span-2" />
        <TextArea label={t('resumeAdmin.highlights')} value={item.highlights.join('\n')} disabled={disabled} onChange={(value) => onChange({ highlights: splitLines(value) })} className="md:col-span-2" />
      </div>
    )}
  </article>
  );
};

const SkillEditor: React.FC<{
  item: ResumeSkill;
  index: number;
  disabled: boolean;
  collapsed: boolean;
  onToggleCollapsed: () => void;
  onChange: (patch: Partial<ResumeSkill>) => void;
  onRemove: () => void;
}> = ({ item, index, disabled, collapsed, onToggleCollapsed, onChange, onRemove }) => {
  const language = usePreferenceStore((state) => state.language);
  const t = (key: Parameters<typeof translate>[1]) => translate(language, key);
  return (
  <article className="border border-mountain-grey bg-paper p-4">
    <EditorHeader title={item.name || formatText(t('resumeAdmin.itemNo'), { value: index + 1 })} collapsed={collapsed} disabled={disabled} onToggleCollapsed={onToggleCollapsed} onRemove={onRemove} />
    {!collapsed && (
      <div className="grid grid-cols-1 gap-4 md:grid-cols-2">
        <TextInput label={t('resumeAdmin.skillCategory')} value={item.category} disabled={disabled} onChange={(category) => onChange({ category })} />
        <TextInput label={t('resumeAdmin.skillName')} value={item.name} disabled={disabled} onChange={(name) => onChange({ name })} />
        <label className="block text-sm text-ink-light">
          <span className="mb-2 block tracking-widest">{t('resumeAdmin.skillLevel')}</span>
          <input
            type="number"
            min={0}
            max={100}
            value={item.level}
            onChange={(event) => onChange({ level: Number(event.target.value) })}
            disabled={disabled}
            className="w-full border border-mountain-grey bg-transparent px-3 py-2 text-ink outline-none focus:border-ochre disabled:cursor-not-allowed disabled:opacity-50"
          />
        </label>
        <TextInput label={t('resumeAdmin.description')} value={item.description} disabled={disabled} onChange={(description) => onChange({ description })} />
      </div>
    )}
  </article>
  );
};

const DateRangeInputs = <T extends { startDate: string; endDate: string }>({
  start,
  end,
  disabled,
  onChange,
}: {
  start: string;
  end: string;
  disabled: boolean;
  onChange: (patch: Partial<T>) => void;
}) => {
  const language = usePreferenceStore((state) => state.language);
  const t = (key: Parameters<typeof translate>[1]) => translate(language, key);
  return (
  <div className="grid grid-cols-2 gap-3">
    <TextInput label={t('resumeAdmin.startDate')} value={start} disabled={disabled} onChange={(startDate) => onChange({ startDate } as Partial<T>)} />
    <TextInput label={t('resumeAdmin.endDate')} value={end} disabled={disabled} onChange={(endDate) => onChange({ endDate } as Partial<T>)} />
  </div>
  );
};

const EditorHeader: React.FC<{
  title: string;
  collapsed: boolean;
  disabled: boolean;
  onToggleCollapsed: () => void;
  onRemove: () => void;
}> = ({ title, collapsed, disabled, onToggleCollapsed, onRemove }) => {
  const language = usePreferenceStore((state) => state.language);
  const t = (key: Parameters<typeof translate>[1]) => translate(language, key);
  return (
  <div className={`${collapsed ? '' : 'mb-4'} flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between`}>
    <h5 className="font-bold tracking-widest text-ink">{title}</h5>
    <div className="flex flex-wrap gap-2">
      <button
        type="button"
        onClick={onToggleCollapsed}
        aria-expanded={!collapsed}
        className="border border-mountain-grey px-3 py-1.5 text-sm text-ink-light transition-colors hover:border-ochre hover:text-ochre"
      >
        {collapsed ? t('resumeAdmin.expand') : t('resumeAdmin.collapse')}
      </button>
      <button
        type="button"
        onClick={onRemove}
        disabled={disabled}
        className="border border-ochre px-3 py-1.5 text-sm text-ochre transition-colors hover:bg-ochre hover:text-paper disabled:cursor-not-allowed disabled:opacity-50"
      >
        {t('common.delete')}
      </button>
    </div>
  </div>
  );
};

type TextInputProps = {
  label: string;
  value: string;
  disabled: boolean;
  onChange: (value: string) => void;
};

const TextInput: React.FC<TextInputProps> = ({ label, value, disabled, onChange }) => (
  <label className="block text-sm text-ink-light">
    <span className="mb-2 block tracking-widest">{label}</span>
    <input
      value={value}
      onChange={(event) => onChange(event.target.value)}
      disabled={disabled}
      className="w-full border border-mountain-grey bg-transparent px-3 py-2 text-ink outline-none focus:border-ochre disabled:cursor-not-allowed disabled:opacity-50"
    />
  </label>
);

const TextArea: React.FC<TextInputProps & { className?: string }> = ({ label, value, disabled, onChange, className = '' }) => (
  <label className={`block text-sm text-ink-light ${className}`.trim()}>
    <span className="mb-2 block tracking-widest">{label}</span>
    <textarea
      value={value}
      onChange={(event) => onChange(event.target.value)}
      disabled={disabled}
      rows={4}
      className="w-full resize-none border border-mountain-grey bg-transparent px-3 py-2 text-ink outline-none focus:border-ochre disabled:cursor-not-allowed disabled:opacity-50"
    />
  </label>
);

const normalizeResumePage = (page: ResumePage, defaultTitle: string): ResumePage => ({
  ...page,
  title: page.title || defaultTitle,
  subtitle: page.subtitle || '',
  experiences: page.experiences || [],
  educations: page.educations || [],
  skills: page.skills || [],
});

const resumeItemKey = <T extends { id?: number }>(prefix: string) => (item: T, index: number) => {
  if (item.id) {
    return `${prefix}:${item.id}`;
  }
  // 新建条目暂无后端 id，按 index 生成稳定 key；不能拼入 item 内容，否则
  // 每输入一个字符 key 就变，React 会卸载并重挂整个 *Editor 子树导致 input 失焦。
  return `${prefix}:new:${index}`;
};

const normalizeResumeForSave = (page: ResumePage) => ({
  title: page.title.trim(),
  subtitle: page.subtitle.trim(),
  contentMarkdown: page.contentMarkdown,
  experiences: page.experiences.map((item, index) => ({
    role: item.role.trim(),
    organization: item.organization.trim(),
    location: item.location.trim(),
    startDate: item.startDate.trim(),
    endDate: item.endDate.trim(),
    description: item.description.trim(),
    highlights: normalizeHighlights(item.highlights),
    displayOrder: index + 1,
  })),
  educations: page.educations.map((item, index) => ({
    school: item.school.trim(),
    degree: item.degree.trim(),
    major: item.major.trim(),
    location: item.location.trim(),
    startDate: item.startDate.trim(),
    endDate: item.endDate.trim(),
    description: item.description.trim(),
    highlights: normalizeHighlights(item.highlights),
    displayOrder: index + 1,
  })),
  skills: page.skills.map((item, index) => ({
    category: item.category.trim(),
    name: item.name.trim(),
    level: Math.min(100, Math.max(0, Number(item.level) || 0)),
    description: item.description.trim(),
    displayOrder: index + 1,
  })),
});

const createEmptyExperience = (): ResumeExperience => ({
  role: '',
  organization: '',
  location: '',
  startDate: '',
  endDate: '',
  description: '',
  highlights: [],
  displayOrder: 0,
});

const createEmptyEducation = (): ResumeEducation => ({
  school: '',
  degree: '',
  major: '',
  location: '',
  startDate: '',
  endDate: '',
  description: '',
  highlights: [],
  displayOrder: 0,
});

const createEmptySkill = (): ResumeSkill => ({
  category: '',
  name: '',
  level: 60,
  description: '',
  displayOrder: 0,
});

const splitLines = (value: string) => value
  .split(/\r?\n/)
  .map((line) => line.trim())
  .filter(Boolean);

const normalizeHighlights = (values: string[]) => splitLines(values.join('\n'));
