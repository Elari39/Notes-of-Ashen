import React, { useEffect, useMemo, useState } from 'react';
import InlineNotice from '../../components/InlineNotice';
import MarkdownRenderer from '../../components/MarkdownRenderer';
import { getAdminResumePage, updateAdminResumePage } from '../../api/siteSettings';
import { usePreferenceStore } from '../../store/preferences';
import type { ResumePage } from '../../types';
import { getErrorMessage } from '../../utils/error';

const emptyResume: ResumePage = {
  title: '简介',
  subtitle: '',
  contentMarkdown: '',
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
        setSaved(res.data);
        setDraft(res.data);
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
      const res = await updateAdminResumePage({
        title: draft.title.trim(),
        subtitle: draft.subtitle.trim(),
        contentMarkdown: draft.contentMarkdown,
      });
      setSaved(res.data);
      setDraft(res.data);
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
        <div className="space-y-6">
          <div className="grid grid-cols-1 gap-5 md:grid-cols-2">
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
          </div>

          <div className="grid min-h-[34rem] grid-cols-1 gap-6 lg:grid-cols-2">
            <section className="flex min-h-[28rem] flex-col border border-mountain-grey p-4">
              <div className="mb-3 text-sm font-bold tracking-widest text-ink">{text.editor}</div>
              <textarea
                value={draft.contentMarkdown}
                onChange={(event) => setDraft((prev) => ({ ...prev, contentMarkdown: event.target.value }))}
                disabled={isSaving}
                placeholder={text.placeholder}
                className="min-h-0 flex-1 resize-none bg-transparent text-ink-light outline-none disabled:cursor-not-allowed disabled:opacity-50"
              />
            </section>
            <section className="min-h-[28rem] overflow-y-auto border border-mountain-grey bg-[var(--paper-soft)] p-4">
              <div className="mb-3 text-sm font-bold tracking-widest text-ink">{text.preview}</div>
              {draft.contentMarkdown.trim() ? (
                <MarkdownRenderer content={draft.contentMarkdown} />
              ) : (
                <p className="py-12 text-center text-sm tracking-[0.2em] text-ink-light">{text.emptyPreview}</p>
              )}
            </section>
          </div>

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

const getResumeAdminLabels = (language: string) => language === 'zh'
  ? {
      title: '简历管理',
      loading: '简历内容加载中...',
      loadError: '简历内容加载失败',
      saveError: '简历内容保存失败',
      saved: '简历内容已保存',
      pageTitle: '页面标题',
      subtitle: '副标题',
      editor: 'Markdown 编辑',
      preview: '实时预览',
      placeholder: '填写简历 Markdown 内容...',
      emptyPreview: '暂无预览内容',
      save: '保存简历',
      saving: '保存中...',
      reset: '重置',
    }
  : {
      title: 'Resume Management',
      loading: 'Loading resume content...',
      loadError: 'Failed to load resume content',
      saveError: 'Failed to save resume content',
      saved: 'Resume content saved',
      pageTitle: 'Page Title',
      subtitle: 'Subtitle',
      editor: 'Markdown Editor',
      preview: 'Live Preview',
      placeholder: 'Write resume Markdown...',
      emptyPreview: 'Nothing to preview yet',
      save: 'Save Resume',
      saving: 'Saving...',
      reset: 'Reset',
    };
