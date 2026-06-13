import React, { useEffect, useMemo, useState } from 'react';
import InlineNotice from '../../components/InlineNotice';
import MarkdownRenderer from '../../components/MarkdownRenderer';
import { getAdminResumePage, updateAdminResumePage } from '../../api/siteSettings';
import { usePreferenceStore } from '../../store/preferences';
import type { ResumeEducation, ResumeExperience, ResumePage, ResumeSkill } from '../../types';
import { getErrorMessage } from '../../utils/error';

const emptyResume: ResumePage = {
  title: '简介',
  subtitle: '',
  contentMarkdown: '',
  experiences: [],
  educations: [],
  skills: [],
};

const AdminResumeContent: React.FC = () => {
  const language = usePreferenceStore((state) => state.language);
  const text = getResumeAdminLabels(language);
  const [saved, setSaved] = useState<ResumePage | null>(null);
  const [draft, setDraft] = useState<ResumePage>(emptyResume);
  const [isLoading, setIsLoading] = useState(true);
  const [isSaving, setIsSaving] = useState(false);
  const [error, setError] = useState('');
  const [notice, setNotice] = useState('');

  useEffect(() => {
    let active = true;
    setIsLoading(true);
    setError('');
    getAdminResumePage()
      .then((res) => {
        if (!active) {
          return;
        }
        const next = normalizeResumePage(res.data);
        setSaved(next);
        setDraft(next);
      })
      .catch((e: unknown) => {
        if (active) {
          setError(getErrorMessage(e, text.loadError));
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
  }, [text.loadError]);

  const hasChanges = useMemo(
    () => Boolean(saved) && JSON.stringify(saved) !== JSON.stringify(draft),
    [draft, saved],
  );

  const handleSave = async () => {
    setError('');
    setNotice('');
    setIsSaving(true);
    try {
      const payload = normalizeResumeForSave(draft);
      const res = await updateAdminResumePage(payload);
      const next = normalizeResumePage(res.data);
      setSaved(next);
      setDraft(next);
      setNotice(text.saved);
    } catch (e: unknown) {
      setError(getErrorMessage(e, text.saveError));
    } finally {
      setIsSaving(false);
    }
  };

  const handleReset = () => {
    if (saved) {
      setDraft(saved);
    }
    setError('');
    setNotice('');
  };

  return (
    <div>
      <div className="mb-8 border-b border-mountain-grey pb-4">
        <h3 className="text-2xl font-bold tracking-widest text-ink">{text.title}</h3>
      </div>

      <InlineNotice message={error} className="mb-6" />
      <InlineNotice message={notice} tone="success" className="mb-6" />

      {isLoading ? (
        <p className="py-12 text-center text-sm tracking-[0.24em] text-ink-light">{text.loading}</p>
      ) : (
        <div className="space-y-8">
          <div className="grid grid-cols-1 gap-5 md:grid-cols-2">
            <TextInput label={text.pageTitle} value={draft.title} disabled={isSaving} onChange={(title) => setDraft((prev) => ({ ...prev, title }))} />
            <TextInput label={text.subtitle} value={draft.subtitle} disabled={isSaving} onChange={(subtitle) => setDraft((prev) => ({ ...prev, subtitle }))} />
          </div>

          <div className="grid min-h-[30rem] grid-cols-1 gap-6 lg:grid-cols-2">
            <section className="flex min-h-[24rem] flex-col border border-mountain-grey p-4">
              <div className="mb-3 text-sm font-bold tracking-widest text-ink">{text.editor}</div>
              <textarea
                value={draft.contentMarkdown}
                onChange={(event) => setDraft((prev) => ({ ...prev, contentMarkdown: event.target.value }))}
                disabled={isSaving}
                placeholder={text.placeholder}
                className="min-h-0 flex-1 resize-none bg-transparent text-ink-light outline-none disabled:cursor-not-allowed disabled:opacity-50"
              />
            </section>
            <section className="min-h-[24rem] overflow-y-auto border border-mountain-grey bg-[var(--paper-soft)] p-4">
              <div className="mb-3 text-sm font-bold tracking-widest text-ink">{text.preview}</div>
              {draft.contentMarkdown.trim() ? (
                <MarkdownRenderer content={draft.contentMarkdown} />
              ) : (
                <p className="py-12 text-center text-sm tracking-[0.2em] text-ink-light">{text.emptyPreview}</p>
              )}
            </section>
          </div>

          <ResumeArraySection
            title={text.experiences}
            addLabel={text.addExperience}
            emptyLabel={text.emptyExperiences}
            disabled={isSaving}
            items={draft.experiences}
            createItem={createEmptyExperience}
            onChange={(experiences) => setDraft((prev) => ({ ...prev, experiences }))}
            renderItem={(item, index, onChange, onRemove) => (
              <ExperienceEditor item={item} index={index} labels={text} disabled={isSaving} onChange={onChange} onRemove={onRemove} />
            )}
          />

          <ResumeArraySection
            title={text.educations}
            addLabel={text.addEducation}
            emptyLabel={text.emptyEducations}
            disabled={isSaving}
            items={draft.educations}
            createItem={createEmptyEducation}
            onChange={(educations) => setDraft((prev) => ({ ...prev, educations }))}
            renderItem={(item, index, onChange, onRemove) => (
              <EducationEditor item={item} index={index} labels={text} disabled={isSaving} onChange={onChange} onRemove={onRemove} />
            )}
          />

          <ResumeArraySection
            title={text.skills}
            addLabel={text.addSkill}
            emptyLabel={text.emptySkills}
            disabled={isSaving}
            items={draft.skills}
            createItem={createEmptySkill}
            onChange={(skills) => setDraft((prev) => ({ ...prev, skills }))}
            renderItem={(item, index, onChange, onRemove) => (
              <SkillEditor item={item} index={index} labels={text} disabled={isSaving} onChange={onChange} onRemove={onRemove} />
            )}
          />

          <div className="flex flex-wrap gap-3">
            <button
              type="button"
              onClick={handleSave}
              disabled={isSaving || !hasChanges}
              className="border border-ink px-4 py-2 text-sm tracking-widest text-ink transition-colors hover:bg-ink hover:text-paper disabled:cursor-not-allowed disabled:opacity-50"
            >
              {isSaving ? text.saving : text.save}
            </button>
            <button
              type="button"
              onClick={handleReset}
              disabled={isSaving || !hasChanges}
              className="border border-mountain-grey px-4 py-2 text-sm tracking-widest text-ink-light transition-colors hover:border-ochre hover:text-ochre disabled:cursor-not-allowed disabled:opacity-50"
            >
              {text.reset}
            </button>
          </div>
        </div>
      )}
    </div>
  );
};

export default AdminResumeContent;

type Labels = ReturnType<typeof getResumeAdminLabels>;

type ResumeArraySectionProps<T> = {
  title: string;
  addLabel: string;
  emptyLabel: string;
  disabled: boolean;
  items: T[];
  createItem: () => T;
  onChange: (items: T[]) => void;
  renderItem: (item: T, index: number, onChange: (patch: Partial<T>) => void, onRemove: () => void) => React.ReactNode;
};

const ResumeArraySection = <T,>({
  title,
  addLabel,
  emptyLabel,
  disabled,
  items,
  createItem,
  onChange,
  renderItem,
}: ResumeArraySectionProps<T>) => (
  <section className="border border-mountain-grey bg-[var(--paper-soft)] p-4 md:p-5">
    <div className="mb-5 flex flex-col gap-3 md:flex-row md:items-center md:justify-between">
      <h4 className="text-base font-bold tracking-widest text-ink">{title}</h4>
      <button
        type="button"
        onClick={() => onChange([...items, createItem()])}
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
        {items.map((item, index) => (
          <React.Fragment key={index}>
            {renderItem(
              item,
              index,
              (patch) => onChange(items.map((current, currentIndex) => currentIndex === index ? { ...current, ...patch } : current)),
              () => onChange(items.filter((_, currentIndex) => currentIndex !== index)),
            )}
          </React.Fragment>
        ))}
      </div>
    )}
  </section>
);

const ExperienceEditor: React.FC<{
  item: ResumeExperience;
  index: number;
  labels: Labels;
  disabled: boolean;
  onChange: (patch: Partial<ResumeExperience>) => void;
  onRemove: () => void;
}> = ({ item, index, labels, disabled, onChange, onRemove }) => (
  <article className="border border-mountain-grey bg-paper p-4">
    <EditorHeader title={item.role || labels.itemNo(index + 1)} deleteLabel={labels.delete} disabled={disabled} onRemove={onRemove} />
    <div className="grid grid-cols-1 gap-4 md:grid-cols-2">
      <TextInput label={labels.role} value={item.role} disabled={disabled} onChange={(role) => onChange({ role })} />
      <TextInput label={labels.organization} value={item.organization} disabled={disabled} onChange={(organization) => onChange({ organization })} />
      <TextInput label={labels.location} value={item.location} disabled={disabled} onChange={(location) => onChange({ location })} />
      <DateRangeInputs start={item.startDate} end={item.endDate} labels={labels} disabled={disabled} onChange={onChange} />
      <TextArea label={labels.description} value={item.description} disabled={disabled} onChange={(description) => onChange({ description })} className="md:col-span-2" />
      <TextArea label={labels.highlights} value={item.highlights.join('\n')} disabled={disabled} onChange={(value) => onChange({ highlights: splitLines(value) })} className="md:col-span-2" />
    </div>
  </article>
);

const EducationEditor: React.FC<{
  item: ResumeEducation;
  index: number;
  labels: Labels;
  disabled: boolean;
  onChange: (patch: Partial<ResumeEducation>) => void;
  onRemove: () => void;
}> = ({ item, index, labels, disabled, onChange, onRemove }) => (
  <article className="border border-mountain-grey bg-paper p-4">
    <EditorHeader title={item.school || labels.itemNo(index + 1)} deleteLabel={labels.delete} disabled={disabled} onRemove={onRemove} />
    <div className="grid grid-cols-1 gap-4 md:grid-cols-2">
      <TextInput label={labels.school} value={item.school} disabled={disabled} onChange={(school) => onChange({ school })} />
      <TextInput label={labels.degree} value={item.degree} disabled={disabled} onChange={(degree) => onChange({ degree })} />
      <TextInput label={labels.major} value={item.major} disabled={disabled} onChange={(major) => onChange({ major })} />
      <TextInput label={labels.location} value={item.location} disabled={disabled} onChange={(location) => onChange({ location })} />
      <DateRangeInputs start={item.startDate} end={item.endDate} labels={labels} disabled={disabled} onChange={onChange} />
      <TextArea label={labels.description} value={item.description} disabled={disabled} onChange={(description) => onChange({ description })} className="md:col-span-2" />
      <TextArea label={labels.highlights} value={item.highlights.join('\n')} disabled={disabled} onChange={(value) => onChange({ highlights: splitLines(value) })} className="md:col-span-2" />
    </div>
  </article>
);

const SkillEditor: React.FC<{
  item: ResumeSkill;
  index: number;
  labels: Labels;
  disabled: boolean;
  onChange: (patch: Partial<ResumeSkill>) => void;
  onRemove: () => void;
}> = ({ item, index, labels, disabled, onChange, onRemove }) => (
  <article className="border border-mountain-grey bg-paper p-4">
    <EditorHeader title={item.name || labels.itemNo(index + 1)} deleteLabel={labels.delete} disabled={disabled} onRemove={onRemove} />
    <div className="grid grid-cols-1 gap-4 md:grid-cols-2">
      <TextInput label={labels.skillCategory} value={item.category} disabled={disabled} onChange={(category) => onChange({ category })} />
      <TextInput label={labels.skillName} value={item.name} disabled={disabled} onChange={(name) => onChange({ name })} />
      <label className="block text-sm text-ink-light">
        <span className="mb-2 block tracking-widest">{labels.skillLevel}</span>
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
      <TextInput label={labels.description} value={item.description} disabled={disabled} onChange={(description) => onChange({ description })} />
    </div>
  </article>
);

const DateRangeInputs = <T extends { startDate: string; endDate: string }>({
  start,
  end,
  labels,
  disabled,
  onChange,
}: {
  start: string;
  end: string;
  labels: Labels;
  disabled: boolean;
  onChange: (patch: Partial<T>) => void;
}) => (
  <div className="grid grid-cols-2 gap-3">
    <TextInput label={labels.startDate} value={start} disabled={disabled} onChange={(startDate) => onChange({ startDate } as Partial<T>)} />
    <TextInput label={labels.endDate} value={end} disabled={disabled} onChange={(endDate) => onChange({ endDate } as Partial<T>)} />
  </div>
);

const EditorHeader: React.FC<{
  title: string;
  deleteLabel: string;
  disabled: boolean;
  onRemove: () => void;
}> = ({ title, deleteLabel, disabled, onRemove }) => (
  <div className="mb-4 flex items-center justify-between gap-3">
    <h5 className="font-bold tracking-widest text-ink">{title}</h5>
    <button
      type="button"
      onClick={onRemove}
      disabled={disabled}
      className="border border-ochre px-3 py-1.5 text-sm text-ochre transition-colors hover:bg-ochre hover:text-paper disabled:cursor-not-allowed disabled:opacity-50"
    >
      {deleteLabel}
    </button>
  </div>
);

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

const normalizeResumePage = (page: ResumePage): ResumePage => ({
  ...page,
  title: page.title || emptyResume.title,
  subtitle: page.subtitle || '',
  experiences: page.experiences || [],
  educations: page.educations || [],
  skills: page.skills || [],
});

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

const getResumeAdminLabels = (language: string) => language === 'zh'
  ? {
      title: '简历管理',
      loading: '简历内容加载中...',
      loadError: '简历内容加载失败',
      saveError: '简历内容保存失败',
      saved: '简历内容已保存',
      pageTitle: '页面标题',
      subtitle: '副标题',
      editor: 'Markdown 引言',
      preview: '实时预览',
      placeholder: '填写简历引言 Markdown 内容...',
      emptyPreview: '暂无预览内容',
      save: '保存简历',
      saving: '保存中...',
      reset: '重置',
      experiences: '工作与实习经历',
      educations: '教育背景',
      skills: '技能树',
      addExperience: '新增经历',
      addEducation: '新增教育',
      addSkill: '新增技能',
      emptyExperiences: '还没有经历。',
      emptyEducations: '还没有教育背景。',
      emptySkills: '还没有技能。',
      itemNo: (value: number) => `条目 ${value}`,
      delete: '删除',
      role: '职位 / 角色',
      organization: '公司 / 组织',
      school: '学校',
      degree: '学历 / 学位',
      major: '专业',
      location: '地点',
      startDate: '开始时间',
      endDate: '结束时间',
      description: '描述',
      highlights: '亮点（每行一条）',
      skillCategory: '技能分类',
      skillName: '技能名称',
      skillLevel: '熟练度 0-100',
    }
  : {
      title: 'Resume Management',
      loading: 'Loading resume content...',
      loadError: 'Failed to load resume content',
      saveError: 'Failed to save resume content',
      saved: 'Resume content saved',
      pageTitle: 'Page Title',
      subtitle: 'Subtitle',
      editor: 'Markdown Intro',
      preview: 'Live Preview',
      placeholder: 'Write resume intro Markdown...',
      emptyPreview: 'Nothing to preview yet',
      save: 'Save Resume',
      saving: 'Saving...',
      reset: 'Reset',
      experiences: 'Experience',
      educations: 'Education',
      skills: 'Skills',
      addExperience: 'Add Experience',
      addEducation: 'Add Education',
      addSkill: 'Add Skill',
      emptyExperiences: 'No experience yet.',
      emptyEducations: 'No education yet.',
      emptySkills: 'No skills yet.',
      itemNo: (value: number) => `Item ${value}`,
      delete: 'Delete',
      role: 'Role',
      organization: 'Organization',
      school: 'School',
      degree: 'Degree',
      major: 'Major',
      location: 'Location',
      startDate: 'Start',
      endDate: 'End',
      description: 'Description',
      highlights: 'Highlights (one per line)',
      skillCategory: 'Category',
      skillName: 'Skill',
      skillLevel: 'Level 0-100',
    };
