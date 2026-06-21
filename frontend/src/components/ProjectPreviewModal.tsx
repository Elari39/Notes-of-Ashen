import React, { useEffect, useMemo, useRef, useState } from 'react';
import { createPortal } from 'react-dom';
import MarkdownRenderer from './MarkdownRenderer';
import { usePreferenceStore } from '../store/preferences';
import { trapFocus } from '../utils/focusTrap';
import { extractMarkdownHeadings } from '../utils/markdownHeadings';
import type { ProjectItem } from '../types';

type ProjectPreviewModalProps = {
  project: ProjectItem | null;
  onClose: () => void;
};

const HEADING_ID_PREFIX = 'project-preview-';

const ProjectPreviewModal: React.FC<ProjectPreviewModalProps> = ({ project, onClose }) => {
  const language = usePreferenceStore((state) => state.language);
  const isZh = language === 'zh';
  const labels = getModalLabels(language);
  const containerRef = useRef<HTMLDivElement | null>(null);
  const [tocCollapsed, setTocCollapsed] = useState(false);

  const headings = useMemo(
    () => extractMarkdownHeadings(project?.contentMarkdown || '', 3),
    [project?.contentMarkdown],
  );

  useEffect(() => {
    if (!project || typeof document === 'undefined') {
      return undefined;
    }

    const previouslyFocused = document.activeElement as HTMLElement | null;

    const handleKeyDown = (event: KeyboardEvent) => {
      if (event.key === 'Escape') {
        onClose();
        return;
      }
      if (containerRef.current) {
        trapFocus(containerRef.current, event);
      }
    };

    const previousOverflow = document.body.style.overflow;
    document.body.style.overflow = 'hidden';
    document.addEventListener('keydown', handleKeyDown);

    const focusTimer = window.setTimeout(() => {
      const focusable = containerRef.current?.querySelectorAll<HTMLElement>('[data-lightbox-focus]');
      focusable?.[0]?.focus();
    }, 0);

    return () => {
      window.clearTimeout(focusTimer);
      document.body.style.overflow = previousOverflow;
      document.removeEventListener('keydown', handleKeyDown);
      if (previouslyFocused && typeof previouslyFocused.focus === 'function') {
        previouslyFocused.focus();
      }
    };
  }, [project, onClose]);

  if (!project || typeof document === 'undefined') {
    return null;
  }

  const hasContent = project.contentMarkdown.trim().length > 0;
  const showToc = hasContent && headings.length >= 2;

  const handleHeadingClick = (headingId: string) => {
    const target = document.getElementById(`${HEADING_ID_PREFIX}${headingId}`);
    target?.scrollIntoView({ behavior: 'smooth', block: 'start' });
  };

  return createPortal(
    <div
      ref={containerRef}
      className="fixed inset-0 z-50 flex items-start justify-center overflow-y-auto bg-black/60 p-4 md:items-center md:p-8"
      role="dialog"
      aria-modal="true"
      aria-label={isZh ? `项目详情：${project.title}` : `Project details: ${project.title}`}
      onClick={onClose}
    >
      <div
        className="relative my-auto w-full max-w-3xl border border-mountain-grey bg-[var(--paper)] shadow-xl"
        onClick={(event) => event.stopPropagation()}
      >
        <button
          type="button"
          data-lightbox-focus
          onClick={onClose}
          aria-label={labels.close}
          className="absolute right-3 top-3 z-10 inline-flex h-9 w-9 items-center justify-center rounded-full border border-mountain-grey bg-[var(--paper)] text-ink transition-colors hover:border-ochre hover:text-ochre focus:outline-none focus:ring-2 focus:ring-ochre"
        >
          <span aria-hidden="true" className="text-xl leading-none">&times;</span>
        </button>

        {project.coverUrl && (
          <div className="aspect-[4/3] w-full overflow-hidden border-b border-mountain-grey bg-[var(--paper-soft)]">
            <img
              src={project.coverUrl}
              alt={project.title}
              className="h-full w-full object-cover"
              loading="lazy"
            />
          </div>
        )}

        <div className="space-y-5 p-5 md:p-8">
          <header className="space-y-3">
            <div className="flex flex-wrap items-center gap-2">
              <h2 className="text-2xl font-bold tracking-widest text-ink md:text-3xl">{project.title}</h2>
              {project.featured && (
                <span className="border border-ochre px-2 py-0.5 text-[11px] uppercase tracking-[0.18em] text-ochre">
                  {labels.featured}
                </span>
              )}
            </div>
            {(project.role || project.period || project.tags.length > 0) && (
              <div className="flex flex-wrap gap-2 text-xs tracking-[0.16em] text-ink-light">
                {project.role && <span className="border border-mountain-grey px-2 py-1">{project.role}</span>}
                {project.period && <span className="border border-mountain-grey px-2 py-1">{project.period}</span>}
                {project.tags.map((tag) => (
                  <span key={tag} className="border border-mountain-grey px-2 py-1">{tag}</span>
                ))}
              </div>
            )}
            {(project.demoUrl || project.repoUrl) && (
              <div className="flex flex-wrap gap-4 text-sm tracking-widest">
                {project.demoUrl && (
                  <a
                    href={project.demoUrl}
                    target="_blank"
                    rel="noreferrer"
                    className="text-ochre hover:text-ink"
                  >
                    {labels.demo}
                  </a>
                )}
                {project.repoUrl && (
                  <a
                    href={project.repoUrl}
                    target="_blank"
                    rel="noreferrer"
                    className="text-ochre hover:text-ink"
                  >
                    {labels.repo}
                  </a>
                )}
              </div>
            )}
          </header>

          {project.summary && (
            <p className="border-l-2 border-mountain-grey pl-4 text-sm leading-7 tracking-wide text-ink-light">
              {project.summary}
            </p>
          )}

          {showToc && (
            <aside
              className="border border-mountain-grey bg-[var(--paper-soft)] p-4"
              aria-label={labels.tocTitle}
            >
              <div className="mb-3 flex items-center justify-between gap-3">
                <h3 className="text-xs font-bold tracking-[0.2em] text-ink">{labels.tocTitle}</h3>
                <button
                  type="button"
                  onClick={() => setTocCollapsed((prev) => !prev)}
                  className="border border-mountain-grey px-2 py-1 text-xs text-ink-light transition-colors hover:border-ochre hover:text-ochre"
                  aria-expanded={!tocCollapsed}
                >
                  {tocCollapsed ? labels.expand : labels.collapse}
                </button>
              </div>
              {!tocCollapsed && (
                <nav className="max-h-60 space-y-1 overflow-y-auto pr-1 text-sm">
                  {headings.map((heading) => (
                    <button
                      key={heading.id}
                      type="button"
                      onClick={() => handleHeadingClick(heading.id)}
                      className="block w-full truncate border-l border-mountain-grey px-3 py-1.5 text-left text-ink-light transition-colors hover:border-ochre hover:text-ink"
                      style={{ paddingLeft: `${Math.max(0, heading.depth - 1) * 0.65 + 0.75}rem` }}
                      title={heading.title}
                    >
                      {heading.title}
                    </button>
                  ))}
                </nav>
              )}
            </aside>
          )}

          {hasContent ? (
            <MarkdownRenderer
              content={project.contentMarkdown}
              headingIdPrefix={HEADING_ID_PREFIX}
              className="prose-lg"
            />
          ) : (
            <p className="py-10 text-center text-sm tracking-[0.2em] text-ink-light">
              {labels.empty}
            </p>
          )}
        </div>
      </div>
    </div>,
    document.body,
  );
};

const getModalLabels = (language: string) => language === 'zh'
  ? {
      close: '关闭项目详情',
      featured: '精选',
      demo: '演示',
      repo: '代码',
      tocTitle: '目录',
      collapse: '收起目录',
      expand: '展开目录',
      empty: '该项目暂无详情内容',
    }
  : {
      close: 'Close project details',
      featured: 'Featured',
      demo: 'Demo',
      repo: 'Repo',
      tocTitle: 'Contents',
      collapse: 'Collapse',
      expand: 'Expand',
      empty: 'No details for this project yet',
    };

export default ProjectPreviewModal;
