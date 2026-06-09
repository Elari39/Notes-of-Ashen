import React, { useEffect, useState } from 'react';
import InlineNotice from '../components/InlineNotice';
import MarkdownRenderer from '../components/MarkdownRenderer';
import { getProjectsPage } from '../api/siteSettings';
import { usePreferenceStore } from '../store/preferences';
import type { ProjectItem, ProjectsPage } from '../types';
import { getErrorMessage } from '../utils/error';
import { useSEO } from '../utils/seo';

const Projects: React.FC = () => {
  const language = usePreferenceStore((state) => state.language);
  const isZh = language === 'zh';
  const text = getProjectsLabels(language);
  const [page, setPage] = useState<ProjectsPage | null>(null);
  const [isLoading, setIsLoading] = useState(true);
  const [error, setError] = useState('');

  useSEO(page?.title || (isZh ? '项目' : 'Projects'));

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

  return (
    <div className="mx-auto mt-4 w-full max-w-6xl space-y-8 md:mt-8">
      <section className="border-b border-mountain-grey pb-6">
        <p className="mb-3 text-xs uppercase tracking-[0.28em] text-ochre">
          {isZh ? 'PROJECTS' : 'WORKS'}
        </p>
        <h1 className="text-3xl font-bold tracking-widest text-ink md:text-4xl">
          {page?.title || (isZh ? '项目' : 'Projects')}
        </h1>
        {page?.subtitle && (
          <p className="mt-4 max-w-2xl text-sm leading-7 tracking-wide text-ink-light opacity-80">
            {page.subtitle}
          </p>
        )}
      </section>

      {isLoading && (
        <p className="py-12 text-center text-sm tracking-[0.24em] text-ink-light">{text.loading}</p>
      )}
      <InlineNotice message={error} />
      {!isLoading && !error && (!page?.items || page.items.length === 0) && (
        <section className="border border-mountain-grey bg-[var(--paper-soft)] p-6 text-ink-light">
          <p className="text-base leading-8 tracking-wide">{text.empty}</p>
        </section>
      )}
      {!isLoading && !error && page?.items && page.items.length > 0 && (
        <section className="grid grid-cols-1 gap-5 md:grid-cols-2 xl:grid-cols-3">
          {page.items.map((project) => (
            <ProjectCard key={project.id} project={project} labels={text} />
          ))}
        </section>
      )}
    </div>
  );
};

export default Projects;

type ProjectCardProps = {
  project: ProjectItem;
  labels: ReturnType<typeof getProjectsLabels>;
};

const ProjectCard: React.FC<ProjectCardProps> = ({ project, labels }) => (
  <article className="group flex min-h-full flex-col overflow-hidden border border-mountain-grey bg-[var(--paper-soft)] transition-colors hover:border-ochre">
    <div className="relative h-56 overflow-hidden border-b border-mountain-grey bg-[var(--paper)]">
      {project.coverUrl ? (
        <img
          src={project.coverUrl}
          alt={project.title}
          className="h-full w-full object-cover grayscale transition-all duration-500 group-hover:scale-[1.03] group-hover:grayscale-0"
          loading="lazy"
        />
      ) : (
        <div className="flex h-full items-center justify-center text-xs tracking-[0.24em] text-ink-light opacity-70">
          {labels.noCover}
        </div>
      )}
      {project.featured && (
        <span className="absolute left-3 top-3 border border-ochre bg-paper px-2 py-0.5 text-[11px] uppercase tracking-[0.18em] text-ochre">
          {labels.featured}
        </span>
      )}
    </div>
    <div className="flex flex-1 flex-col space-y-5 p-5 md:p-6">
      <div className="flex flex-col gap-3 md:flex-row md:items-start md:justify-between">
        <div>
          <div className="flex flex-wrap items-center gap-2">
            <h2 className="text-xl font-bold tracking-widest text-ink">{project.title}</h2>
          </div>
          {project.summary && (
            <p className="mt-3 text-sm leading-7 tracking-wide text-ink-light">{project.summary}</p>
          )}
        </div>
        {(project.demoUrl || project.repoUrl) && (
          <div className="flex shrink-0 flex-wrap gap-3 text-sm tracking-widest">
            {project.demoUrl && (
              <a href={project.demoUrl} target="_blank" rel="noreferrer" className="text-ochre hover:text-ink">
                {labels.demo}
              </a>
            )}
            {project.repoUrl && (
              <a href={project.repoUrl} target="_blank" rel="noreferrer" className="text-ochre hover:text-ink">
                {labels.repo}
              </a>
            )}
          </div>
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

      <div className="mt-auto">
        {project.contentMarkdown.trim() && (
          <div className="line-clamp-[10]">
            <MarkdownRenderer content={project.contentMarkdown} className="text-sm" />
          </div>
        )}
      </div>
    </div>
  </article>
);

const getProjectsLabels = (language: string) => language === 'zh'
  ? {
      loading: '项目加载中...',
      loadError: '项目内容加载失败',
      empty: '项目内容还没有填写。',
      demo: '演示',
      repo: '代码',
      featured: '精选',
      noCover: '暂无封面',
    }
  : {
      loading: 'Loading projects...',
      loadError: 'Failed to load projects',
      empty: 'No projects have been added yet.',
      demo: 'Demo',
      repo: 'Repo',
      featured: 'Featured',
      noCover: 'No Cover',
    };
