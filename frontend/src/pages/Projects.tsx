import React, { useEffect, useRef, useState } from 'react';
import InlineNotice from '../components/InlineNotice';
import PagePendingState from '../components/RoutePending';
import Skeleton from '../components/Skeleton';
import EmptyState from '../components/ui/EmptyState';
import Tag from '../components/ui/Tag';
import ProjectPreviewModal from '../components/ProjectPreviewModal';
import { getProjectsPage } from '../api/siteSettings';
import { usePreferenceStore } from '../store/preferences';
import { formatText, translate } from '../i18n';
import type { ProjectItem, ProjectsPage } from '../types';
import { getErrorMessage } from '../utils/error';
import { useSEO } from '../utils/seo';

const Projects: React.FC = () => {
  const language = usePreferenceStore((state) => state.language);
  const t = (key: Parameters<typeof translate>[1]) => translate(language, key);
  // 用 ref 持有最新 language，避免切换语言时重新拉取项目数据（数据与 UI 语言无关，
  // language 仅用于错误兜底文案）。
  const languageRef = useRef(language);
  languageRef.current = language;
  const [page, setPage] = useState<ProjectsPage | null>(null);
  const [isLoading, setIsLoading] = useState(true);
  const [error, setError] = useState('');
  const [selected, setSelected] = useState<ProjectItem | null>(null);

  useSEO(page?.title || t('projects.pageTitleFallback'));

  useEffect(() => {
    let active = true;
    setIsLoading(true);
    setError('');
    getProjectsPage()
      .then((res) => {
        if (active) {
          setPage(res.data);
        }
      })
      .catch((e: unknown) => {
        if (active) {
          setError(getErrorMessage(e, translate(languageRef.current, 'projects.loadError')));
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

  return (
    <div className="mx-auto mt-4 w-full max-w-6xl space-y-8 md:mt-8">
      <section className="border-b border-mountain-grey pb-6">
        <p className="mb-3 text-xs uppercase tracking-[0.28em] text-ochre">
          {t('projects.kicker')}
        </p>
        <h1 className="text-3xl font-bold tracking-widest text-ink md:text-4xl">
          {page?.title || t('projects.pageTitleFallback')}
        </h1>
        {page?.subtitle && (
          <p className="mt-4 max-w-2xl text-sm leading-7 tracking-wide text-ink-light opacity-80">
            {page.subtitle}
          </p>
        )}
      </section>

      {isLoading && !page && (
        <section className="grid grid-cols-1 gap-5 md:grid-cols-2 xl:grid-cols-3" aria-hidden="true">
          {Array.from({ length: 6 }).map((_, idx) => (
            <div key={idx} className="flex min-h-full flex-col overflow-hidden border border-mountain-grey bg-[var(--paper-soft)]">
              <Skeleton className="aspect-[4/3] w-full border-0 border-b border-mountain-grey" />
              <div className="flex flex-1 flex-col gap-4 p-5">
                <Skeleton className="h-6 w-3/4" />
                <Skeleton className="h-3 w-full" />
                <Skeleton className="h-3 w-5/6" />
              </div>
            </div>
          ))}
        </section>
      )}
      {isLoading && page && (
        <PagePendingState variant="inline" label={t('projects.loading')} />
      )}
      <InlineNotice message={error} />
      {!isLoading && !error && (!page?.items || page.items.length === 0) && (
        <EmptyState illustration="leaf" title={t('projects.empty')} />
      )}
      {!isLoading && !error && page?.items && page.items.length > 0 && (
        <section className="grid grid-cols-1 gap-5 md:grid-cols-2 xl:grid-cols-3">
          {page.items.map((project) => (
            <ProjectCard key={project.id} project={project} onSelect={setSelected} />
          ))}
        </section>
      )}

      <ProjectPreviewModal project={selected} onClose={() => setSelected(null)} />
    </div>
  );
};

export default Projects;

type ProjectCardProps = {
  project: ProjectItem;
  onSelect: (project: ProjectItem) => void;
};

const ProjectCard: React.FC<ProjectCardProps> = React.memo(({ project, onSelect }) => {
  const language = usePreferenceStore((state) => state.language);
  const t = (key: Parameters<typeof translate>[1]) => translate(language, key);
  const handleKeyDown = (event: React.KeyboardEvent<HTMLElement>) => {
    if (event.key === 'Enter' || event.key === ' ') {
      event.preventDefault();
      onSelect(project);
    }
  };

  return (
    <article
      role="button"
      tabIndex={0}
      aria-label={formatText(t('projects.openDetail'), { title: project.title })}
      onClick={() => onSelect(project)}
      onKeyDown={handleKeyDown}
      className="group flex min-h-full cursor-pointer flex-col overflow-hidden border border-mountain-grey bg-[var(--paper-soft)] outline-none transition-colors hover:border-ochre focus-visible:border-ochre focus-visible:ring-2 focus-visible:ring-ochre"
    >
      <div className="relative aspect-[4/3] overflow-hidden border-b border-mountain-grey bg-[var(--paper)]">
        {project.coverUrl ? (
          <img
            src={project.coverUrl}
            alt={project.title}
            className="h-full w-full object-cover grayscale transition-[transform,filter] duration-slow ease-paper group-hover:scale-[1.03] group-hover:grayscale-0"
            loading="lazy"
          />
        ) : (
          <div className="flex h-full items-center justify-center text-xs tracking-[0.24em] text-ink-light opacity-70">
            {t('projects.noCover')}
          </div>
        )}
        {project.featured && (
          <span className="absolute left-3 top-3 z-10">
            <Tag tone="ochre" size="sm">{t('projects.featured')}</Tag>
          </span>
        )}
      </div>
      <div className="flex flex-1 flex-col gap-5 p-5 md:p-6">
        <div className="flex flex-1 flex-col gap-3">
          <div className="flex flex-col gap-3 sm:flex-row sm:items-start sm:justify-between">
            <div className="flex flex-wrap items-center gap-2">
              <h2 className="text-xl font-bold tracking-widest text-ink">{project.title}</h2>
            </div>
            {(project.demoUrl || project.repoUrl) && (
              <div className="flex shrink-0 flex-wrap gap-3 text-sm tracking-widest" onClick={(event) => event.stopPropagation()}>
                {project.demoUrl && (
                  <a href={project.demoUrl} target="_blank" rel="noreferrer" className="text-ochre hover:text-ink">
                    {t('projects.demo')}
                  </a>
                )}
                {project.repoUrl && (
                  <a href={project.repoUrl} target="_blank" rel="noreferrer" className="text-ochre hover:text-ink">
                    {t('projects.repo')}
                  </a>
                )}
              </div>
            )}
          </div>
          {project.summary && (
            <p className="line-clamp-5 text-sm leading-7 tracking-wide text-ink-light">{project.summary}</p>
          )}
        </div>

        {(project.role || project.period || project.tags.length > 0) && (
          <div className="flex flex-wrap gap-2 text-xs tracking-[0.16em] text-ink-light">
            {project.role && <Tag tone="neutral" size="sm">{project.role}</Tag>}
            {project.period && <Tag tone="neutral" size="sm">{project.period}</Tag>}
            {project.tags.map((tag) => (
              <Tag key={tag} tone="neutral" size="sm">{tag}</Tag>
            ))}
          </div>
        )}

        <div className="mt-auto border-t border-mountain-grey pt-4 text-xs tracking-[0.2em] text-ochre transition-colors group-hover:text-ink">
          {t('projects.viewDetail')}
        </div>
      </div>
    </article>
  );
});
ProjectCard.displayName = 'ProjectCard';
