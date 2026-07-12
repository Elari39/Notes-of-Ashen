import React, { useEffect, useRef, useState } from 'react';
import InlineNotice from '../components/InlineNotice';
import PagePendingState from '../components/RoutePending';
import Skeleton from '../components/Skeleton';
import EmptyState from '../components/ui/EmptyState';
import Button from '../components/ui/Button';
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
  const [retryVersion, setRetryVersion] = useState(0);

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
  }, [retryVersion]);

  const handleRetry = () => {
    setRetryVersion((version) => version + 1);
  };

  return (
    <div className="editorial-container w-full space-y-10">
      <section className="max-w-3xl py-6 md:py-10">
        <p className="mb-3 text-xs uppercase tracking-[0.28em] text-ochre">
          {t('projects.kicker')}
        </p>
        <h1 className="editorial-page-title">
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
            <div key={idx} className="flex min-h-full flex-col overflow-hidden rounded-lg bg-surface-card">
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
      <InlineNotice
        message={error}
        action={(
          <Button type="button" variant="ghost" size="sm" onClick={handleRetry}>
            {t('common.retry')}
          </Button>
        )}
      />
      {!isLoading && page && page.items.length === 0 && (
        <EmptyState illustration="leaf" title={t('projects.empty')} />
      )}
      {!isLoading && page && page.items.length > 0 && (
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

  return (
    <article
      className="group flex min-h-full flex-col overflow-hidden rounded-lg bg-surface-card transition-colors hover:bg-surface-strong"
    >
      <div className="relative aspect-[4/3] overflow-hidden bg-paper">
        {project.coverUrl ? (
          <img
            src={project.coverUrl}
            alt={project.title}
            className="h-full w-full object-cover opacity-90 transition-[transform,opacity] duration-slow ease-paper group-hover:scale-[1.025] group-hover:opacity-100"
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
              <h2 className="font-display text-3xl leading-tight text-ink">{project.title}</h2>
            </div>
            {(project.demoUrl || project.repoUrl) && (
              <div className="flex shrink-0 flex-wrap gap-3 text-sm tracking-widest">
                {project.demoUrl && (
                  <a href={project.demoUrl} target="_blank" rel="noreferrer" className="text-ochre transition-colors hover:text-ink focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-ochre">
                    {t('projects.demo')}
                  </a>
                )}
                {project.repoUrl && (
                  <a href={project.repoUrl} target="_blank" rel="noreferrer" className="text-ochre transition-colors hover:text-ink focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-ochre">
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

        <div className="mt-auto border-t border-mountain-grey pt-4">
          <Button
            type="button"
            variant="link"
            size="sm"
            onClick={() => onSelect(project)}
            aria-label={formatText(t('projects.openDetail'), { title: project.title })}
          >
            {t('projects.viewDetail')}
          </Button>
        </div>
      </div>
    </article>
  );
});
ProjectCard.displayName = 'ProjectCard';
