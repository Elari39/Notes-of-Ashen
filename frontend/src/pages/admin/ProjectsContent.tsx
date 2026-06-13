import React, { useEffect, useMemo, useState } from 'react';
import InlineNotice from '../../components/InlineNotice';
import MarkdownRenderer from '../../components/MarkdownRenderer';
import { getAdminProjectsPage, updateAdminProjectsPage } from '../../api/siteSettings';
import { getTags } from '../../api/tag';
import { usePreferenceStore } from '../../store/preferences';
import type { ProjectItem, ProjectsPage, Tag } from '../../types';
import { getErrorMessage } from '../../utils/error';

const emptyProjects: ProjectsPage = {
  title: '项目',
  subtitle: '',
  items: [],
};

const AdminProjectsContent: React.FC = () => {
  const language = usePreferenceStore((state) => state.language);
  const text = getProjectsAdminLabels(language);
  const [saved, setSaved] = useState<ProjectsPage | null>(null);
  const [draft, setDraft] = useState<ProjectsPage>(emptyProjects);
  const [isLoading, setIsLoading] = useState(true);
  const [isSaving, setIsSaving] = useState(false);
  const [error, setError] = useState('');
  const [notice, setNotice] = useState('');
  const [allTags, setAllTags] = useState<Tag[]>([]);
  const [collapsedProjectIds, setCollapsedProjectIds] = useState<Set<string>>(() => new Set());

  useEffect(() => {
    let active = true;
    setIsLoading(true);
    setError('');
    Promise.all([getAdminProjectsPage(), getTags({ page: 1, size: 100 })])
      .then(([res, tagsRes]) => {
        if (!active) {
          return;
        }
        const next = normalizeProjectsPage(res.data);
        setSaved(next);
        setDraft(next);
        setAllTags(tagsRes.data.items);
        setCollapsedProjectIds(new Set());
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

  const addProject = () => {
    setDraft((prev) => ({
      ...prev,
      items: [...prev.items, createEmptyProject()],
    }));
    setNotice('');
  };

  const updateProject = (index: number, patch: Partial<ProjectItem>) => {
    setDraft((prev) => ({
      ...prev,
      items: prev.items.map((item, itemIndex) => itemIndex === index ? { ...item, ...patch } : item),
    }));
  };

  const moveProject = (index: number, direction: -1 | 1) => {
    setDraft((prev) => {
      const nextIndex = index + direction;
      if (nextIndex < 0 || nextIndex >= prev.items.length) {
        return prev;
      }
      const items = [...prev.items];
      const current = items[index];
      items[index] = items[nextIndex];
      items[nextIndex] = current;
      return { ...prev, items };
    });
  };

  const removeProject = (index: number) => {
    if (!window.confirm(text.confirmDelete)) {
      return;
    }
    const projectId = draft.items[index]?.id;
    setDraft((prev) => ({
      ...prev,
      items: prev.items.filter((_, itemIndex) => itemIndex !== index),
    }));
    if (projectId) {
      setCollapsedProjectIds((prev) => {
        const next = new Set(prev);
        next.delete(projectId);
        return next;
      });
    }
  };

  const toggleProjectCollapsed = (projectId: string) => {
    setCollapsedProjectIds((prev) => {
      const next = new Set(prev);
      if (next.has(projectId)) {
        next.delete(projectId);
      } else {
        next.add(projectId);
      }
      return next;
    });
  };

  const handleSave = async () => {
    setError('');
    setNotice('');
    setIsSaving(true);
    try {
      const res = await updateAdminProjectsPage({
        title: draft.title.trim(),
        subtitle: draft.subtitle.trim(),
        items: draft.items.map((item) => ({
          ...item,
          id: item.id.trim(),
          title: item.title.trim(),
          summary: item.summary.trim(),
          role: item.role.trim(),
          period: item.period.trim(),
          tags: normalizeTags(item.tags),
          tagIds: normalizeTagIds(item.tagIds || []),
          coverUrl: item.coverUrl.trim(),
          demoUrl: item.demoUrl.trim(),
          repoUrl: item.repoUrl.trim(),
        })),
      });
      const next = normalizeProjectsPage(res.data);
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
      <div className="mb-8 flex flex-col gap-4 border-b border-mountain-grey pb-4 md:flex-row md:items-center md:justify-between">
        <h3 className="text-2xl font-bold tracking-widest text-ink">{text.title}</h3>
        {!isLoading && (
          <button
            type="button"
            onClick={addProject}
            disabled={isSaving}
            className="border border-ochre px-4 py-2 text-sm tracking-widest text-ochre transition-colors hover:bg-ochre hover:text-paper disabled:cursor-not-allowed disabled:opacity-50"
          >
            {text.add}
          </button>
        )}
      </div>

      <InlineNotice message={error} className="mb-6" />
      <InlineNotice message={notice} tone="success" className="mb-6" />

      {isLoading ? (
        <p className="py-12 text-center text-sm tracking-[0.24em] text-ink-light">{text.loading}</p>
      ) : (
        <div className="space-y-6">
          <section className="grid grid-cols-1 gap-5 md:grid-cols-2">
            <label className="block text-sm text-ink-light">
              <span className="mb-2 block tracking-widest">{text.pageTitle}</span>
              <input
                value={draft.title}
                onChange={(event) => setDraft((prev) => ({ ...prev, title: event.target.value }))}
                disabled={isSaving}
                className="w-full border border-mountain-grey bg-transparent px-3 py-2 text-ink outline-none focus:border-ochre disabled:cursor-not-allowed disabled:opacity-50"
              />
            </label>
            <label className="block text-sm text-ink-light">
              <span className="mb-2 block tracking-widest">{text.subtitle}</span>
              <input
                value={draft.subtitle}
                onChange={(event) => setDraft((prev) => ({ ...prev, subtitle: event.target.value }))}
                disabled={isSaving}
                className="w-full border border-mountain-grey bg-transparent px-3 py-2 text-ink outline-none focus:border-ochre disabled:cursor-not-allowed disabled:opacity-50"
              />
            </label>
          </section>

          {draft.items.length === 0 ? (
            <section className="border border-mountain-grey bg-[var(--paper-soft)] p-6 text-center">
              <p className="text-sm tracking-[0.2em] text-ink-light">{text.empty}</p>
              <button
                type="button"
                onClick={addProject}
                disabled={isSaving}
                className="mt-5 border border-ink px-4 py-2 text-sm tracking-widest text-ink transition-colors hover:bg-ink hover:text-paper disabled:cursor-not-allowed disabled:opacity-50"
              >
                {text.add}
              </button>
            </section>
          ) : (
            <section className="space-y-5">
              {draft.items.map((project, index) => (
                <ProjectEditor
                  key={project.id}
                  project={project}
                  index={index}
                  total={draft.items.length}
                  labels={text}
                  allTags={allTags}
                  disabled={isSaving}
                  collapsed={collapsedProjectIds.has(project.id)}
                  onChange={(patch) => updateProject(index, patch)}
                  onMove={(direction) => moveProject(index, direction)}
                  onRemove={() => removeProject(index)}
                  onToggleCollapsed={() => toggleProjectCollapsed(project.id)}
                />
              ))}
            </section>
          )}

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

export default AdminProjectsContent;

type ProjectEditorProps = {
  project: ProjectItem;
  index: number;
  total: number;
  labels: ReturnType<typeof getProjectsAdminLabels>;
  allTags: Tag[];
  disabled: boolean;
  collapsed: boolean;
  onChange: (patch: Partial<ProjectItem>) => void;
  onMove: (direction: -1 | 1) => void;
  onRemove: () => void;
  onToggleCollapsed: () => void;
};

const ProjectEditor: React.FC<ProjectEditorProps> = ({
  project,
  index,
  total,
  labels,
  allTags,
  disabled,
  collapsed,
  onChange,
  onMove,
  onRemove,
  onToggleCollapsed,
}) => (
  <article className="border border-mountain-grey bg-[var(--paper-soft)] p-4 md:p-5">
    <div className={`${collapsed ? '' : 'mb-5'} flex flex-col gap-3 md:flex-row md:items-center md:justify-between`}>
      <div>
        <p className="text-xs tracking-[0.2em] text-ochre">{labels.projectNo(index + 1)}</p>
        <h4 className="mt-1 text-base font-bold tracking-widest text-ink">
          {project.title.trim() || labels.untitled}
        </h4>
      </div>
      <div className="flex flex-wrap gap-2">
        <button
          type="button"
          onClick={onToggleCollapsed}
          aria-expanded={!collapsed}
          className="border border-mountain-grey px-3 py-1.5 text-sm text-ink-light transition-colors hover:border-ochre hover:text-ochre"
        >
          {collapsed ? labels.expand : labels.collapse}
        </button>
        <button
          type="button"
          onClick={() => onMove(-1)}
          disabled={disabled || index === 0}
          className="border border-mountain-grey px-3 py-1.5 text-sm text-ink-light transition-colors hover:border-ochre hover:text-ochre disabled:cursor-not-allowed disabled:opacity-40"
        >
          {labels.up}
        </button>
        <button
          type="button"
          onClick={() => onMove(1)}
          disabled={disabled || index === total - 1}
          className="border border-mountain-grey px-3 py-1.5 text-sm text-ink-light transition-colors hover:border-ochre hover:text-ochre disabled:cursor-not-allowed disabled:opacity-40"
        >
          {labels.down}
        </button>
        <button
          type="button"
          onClick={onRemove}
          disabled={disabled}
          className="border border-ochre px-3 py-1.5 text-sm text-ochre transition-colors hover:bg-ochre hover:text-paper disabled:cursor-not-allowed disabled:opacity-50"
        >
          {labels.delete}
        </button>
      </div>
    </div>

    {!collapsed && (
      <>
        <div className="grid grid-cols-1 gap-4 md:grid-cols-2">
          <TextInput label={labels.projectTitle} value={project.title} disabled={disabled} onChange={(value) => onChange({ title: value })} />
          <TextInput label={labels.role} value={project.role} disabled={disabled} onChange={(value) => onChange({ role: value })} />
          <TextInput label={labels.period} value={project.period} disabled={disabled} onChange={(value) => onChange({ period: value })} />
          <TextInput label={labels.coverUrl} value={project.coverUrl} disabled={disabled} onChange={(value) => onChange({ coverUrl: value })} />
          <TextInput label={labels.demoUrl} value={project.demoUrl} disabled={disabled} onChange={(value) => onChange({ demoUrl: value })} />
          <TextInput label={labels.repoUrl} value={project.repoUrl} disabled={disabled} onChange={(value) => onChange({ repoUrl: value })} />
          <label className="flex items-center gap-3 border border-mountain-grey px-3 py-2 text-sm text-ink-light">
            <input
              type="checkbox"
              checked={project.featured}
              onChange={(event) => onChange({ featured: event.target.checked })}
              disabled={disabled}
              className="h-4 w-4 accent-ochre disabled:cursor-not-allowed"
            />
            <span className="tracking-widest">{labels.featured}</span>
          </label>
          <TagSelector
            label={labels.tags}
            tags={allTags}
            selectedIds={project.tagIds || []}
            fallbackNames={project.tags}
            disabled={disabled}
            onChange={(tagIds) => onChange({ tagIds })}
          />
          <label className="block text-sm text-ink-light md:col-span-2">
            <span className="mb-2 block tracking-widest">{labels.summary}</span>
            <textarea
              value={project.summary}
              onChange={(event) => onChange({ summary: event.target.value })}
              disabled={disabled}
              rows={3}
              className="w-full resize-none border border-mountain-grey bg-transparent px-3 py-2 text-ink outline-none focus:border-ochre disabled:cursor-not-allowed disabled:opacity-50"
            />
          </label>
        </div>

        {project.coverUrl && (
          <img
            src={project.coverUrl}
            alt={project.title}
            className="mt-5 max-h-64 w-full border border-mountain-grey object-cover"
            loading="lazy"
          />
        )}

        <div className="mt-5 grid min-h-[24rem] grid-cols-1 gap-5 lg:grid-cols-2">
          <section className="flex min-h-[20rem] flex-col border border-mountain-grey p-4">
            <div className="mb-3 text-sm font-bold tracking-widest text-ink">{labels.detailEditor}</div>
            <textarea
              value={project.contentMarkdown}
              onChange={(event) => onChange({ contentMarkdown: event.target.value })}
              disabled={disabled}
              placeholder={labels.detailPlaceholder}
              className="min-h-0 flex-1 resize-none bg-transparent text-ink-light outline-none disabled:cursor-not-allowed disabled:opacity-50"
            />
          </section>
          <section className="min-h-[20rem] overflow-y-auto border border-mountain-grey bg-paper p-4">
            <div className="mb-3 text-sm font-bold tracking-widest text-ink">{labels.preview}</div>
            {project.contentMarkdown.trim() ? (
              <MarkdownRenderer content={project.contentMarkdown} />
            ) : (
              <p className="py-12 text-center text-sm tracking-[0.2em] text-ink-light">{labels.emptyPreview}</p>
            )}
          </section>
        </div>
      </>
    )}
  </article>
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

const TagSelector: React.FC<{
  label: string;
  tags: Tag[];
  selectedIds: number[];
  fallbackNames: string[];
  disabled: boolean;
  onChange: (tagIds: number[]) => void;
}> = ({ label, tags, selectedIds, fallbackNames, disabled, onChange }) => (
  <div className="block text-sm text-ink-light md:col-span-2">
    <span className="mb-2 block tracking-widest">{label}</span>
    {tags.length === 0 ? (
      <p className="border border-mountain-grey px-3 py-2 text-xs tracking-[0.16em] text-ink-light opacity-80">
        {fallbackNames.length > 0 ? fallbackNames.join(', ') : 'No tags'}
      </p>
    ) : (
      <div className="flex flex-wrap gap-2">
        {tags.map((tag) => {
          const checked = selectedIds.includes(tag.id);
          return (
            <label
              key={tag.id}
              className={`inline-flex items-center gap-2 border px-3 py-1.5 text-xs tracking-[0.14em] transition-colors ${
                checked ? 'border-ochre text-ochre' : 'border-mountain-grey text-ink-light'
              }`}
            >
              <input
                type="checkbox"
                checked={checked}
                disabled={disabled}
                onChange={(event) => {
                  if (event.target.checked) {
                    onChange([...selectedIds, tag.id]);
                  } else {
                    onChange(selectedIds.filter((id) => id !== tag.id));
                  }
                }}
                className="h-4 w-4 accent-ochre disabled:cursor-not-allowed"
              />
              {tag.name}
            </label>
          );
        })}
      </div>
    )}
  </div>
);

const createEmptyProject = (): ProjectItem => ({
  id: `project-${Date.now()}-${Math.random().toString(36).slice(2, 8)}`,
  tagIds: [],
  title: '',
  summary: '',
  role: '',
  period: '',
  tags: [],
  coverUrl: '',
  demoUrl: '',
  repoUrl: '',
  contentMarkdown: '',
  featured: false,
});

const normalizeTags = (tags: string[]) => {
  const seen = new Set<string>();
  return tags
    .map((tag) => tag.trim())
    .filter((tag) => {
      if (!tag) {
        return false;
      }
      const key = tag.toLowerCase();
      if (seen.has(key)) {
        return false;
      }
      seen.add(key);
      return true;
    });
};

const normalizeTagIds = (tagIds: number[]) => Array.from(new Set(tagIds.filter((id) => id > 0)));

const normalizeProjectsPage = (page: ProjectsPage): ProjectsPage => ({
  ...page,
  title: page.title || emptyProjects.title,
  subtitle: page.subtitle || '',
  items: page.items || [],
});

const getProjectsAdminLabels = (language: string) => language === 'zh'
  ? {
      title: '项目管理',
      loading: '项目内容加载中...',
      loadError: '项目内容加载失败',
      saveError: '项目内容保存失败',
      saved: '项目内容已保存',
      pageTitle: '页面标题',
      subtitle: '副标题',
      empty: '还没有项目。',
      add: '新增项目',
      save: '保存项目',
      saving: '保存中...',
      reset: '重置',
      confirmDelete: '确认删除这个项目？',
      projectNo: (value: number) => `项目 ${value}`,
      untitled: '未命名项目',
      up: '上移',
      down: '下移',
      collapse: '收起',
      expand: '展开',
      delete: '删除',
      projectTitle: '项目标题',
      role: '角色',
      period: '周期',
      tags: '技术标签',
      coverUrl: '封面 URL',
      demoUrl: '演示 URL',
      repoUrl: '代码仓库 URL',
      featured: '精选项目',
      summary: '摘要',
      detailEditor: '详情 Markdown',
      detailPlaceholder: '填写项目详情 Markdown...',
      preview: '详情预览',
      emptyPreview: '暂无预览内容',
    }
  : {
      title: 'Project Management',
      loading: 'Loading project content...',
      loadError: 'Failed to load project content',
      saveError: 'Failed to save project content',
      saved: 'Project content saved',
      pageTitle: 'Page Title',
      subtitle: 'Subtitle',
      empty: 'No projects yet.',
      add: 'Add Project',
      save: 'Save Projects',
      saving: 'Saving...',
      reset: 'Reset',
      confirmDelete: 'Delete this project?',
      projectNo: (value: number) => `Project ${value}`,
      untitled: 'Untitled Project',
      up: 'Up',
      down: 'Down',
      collapse: 'Collapse',
      expand: 'Expand',
      delete: 'Delete',
      projectTitle: 'Project Title',
      role: 'Role',
      period: 'Period',
      tags: 'Tags',
      coverUrl: 'Cover URL',
      demoUrl: 'Demo URL',
      repoUrl: 'Repository URL',
      featured: 'Featured Project',
      summary: 'Summary',
      detailEditor: 'Detail Markdown',
      detailPlaceholder: 'Write project detail Markdown...',
      preview: 'Detail Preview',
      emptyPreview: 'Nothing to preview yet',
    };
