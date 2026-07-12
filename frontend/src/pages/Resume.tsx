import React, { useEffect, useMemo, useRef, useState } from 'react';
import { GraphChart } from 'echarts/charts';
import { TooltipComponent } from 'echarts/components';
import { init, use } from 'echarts/core';
import { useNavigate } from 'react-router-dom';
import { CanvasRenderer } from 'echarts/renderers';
import type { ECharts, EChartsCoreOption } from 'echarts/core';
import InlineNotice from '../components/InlineNotice';
import MarkdownRenderer from '../components/MarkdownRenderer';
import PagePendingState from '../components/RoutePending';
import EmptyState from '../components/ui/EmptyState';
import Button from '../components/ui/Button';
import { getProjectsPage, getResumePage } from '../api/siteSettings';
import { getTags } from '../api/tag';
import { usePreferenceStore } from '../store/preferences';
import { translate } from '../i18n';
import type { ProjectItem, ResumeEducation, ResumeExperience, ResumePage, ResumeSkill, Tag } from '../types';
import { getErrorMessage } from '../utils/error';
import { useSEO } from '../utils/seo';

use([GraphChart, TooltipComponent, CanvasRenderer]);

const Resume: React.FC = () => {
  const language = usePreferenceStore((state) => state.language);
  const effectiveTheme = usePreferenceStore((state) => state.effectiveTheme);
  const accentColor = usePreferenceStore((state) => state.accentColor);
  const t = (key: Parameters<typeof translate>[1]) => translate(language, key);
  // 用 ref 持有最新 language，避免切换语言时重新拉取简历正文（正文与语言无关）。
  const languageRef = useRef(language);
  languageRef.current = language;
  const navigate = useNavigate();
  const [page, setPage] = useState<ResumePage | null>(null);
  const [projects, setProjects] = useState<ProjectItem[]>([]);
  const [tags, setTags] = useState<Tag[]>([]);
  const [isLoading, setIsLoading] = useState(true);
  const [isExporting, setIsExporting] = useState(false);
  const [error, setError] = useState('');
  const [exportError, setExportError] = useState('');
  const [retryVersion, setRetryVersion] = useState(0);
  const resumeRef = useRef<HTMLDivElement>(null);

  useSEO(page?.title || t('resume.pageTitleFallback'));

  useEffect(() => {
    let active = true;
    setIsLoading(true);
    setError('');
    getResumePage()
      .then((res) => {
        if (active) {
          setPage(normalizeResumePage(res.data));
        }
      })
      .catch((e: unknown) => {
        if (active) {
          setError(getErrorMessage(e, translate(languageRef.current, 'resume.loadError')));
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

  useEffect(() => {
    let active = true;
    Promise.allSettled([
      getProjectsPage(),
      getTags({ page: 1, size: 100 }),
    ]).then(([projectsRes, tagsRes]) => {
      if (!active) {
        return;
      }
      if (projectsRes.status === 'fulfilled') {
        setProjects(projectsRes.value.data.items || []);
      }
      if (tagsRes.status === 'fulfilled') {
        setTags(tagsRes.value.data.items || []);
      }
    });
    return () => {
      active = false;
    };
  }, []);

  const hasContent = Boolean(
    page?.contentMarkdown.trim()
    || page?.experiences.length
    || page?.educations.length
    || page?.skills.length,
  );

  const skillsByCategory = useMemo(() => groupSkills(page?.skills || []), [page?.skills]);
  const skillGraph = useMemo(
    () => buildSkillGraph(page?.skills || [], projects, tags),
    [page?.skills, projects, tags],
  );

  const handleExportPDF = async () => {
    if (!resumeRef.current || !page || isExporting) {
      return;
    }
    setExportError('');
    setIsExporting(true);
    resumeRef.current.classList.add('resume-pdf-exporting');
    try {
      const { default: html2pdf } = await import('html2pdf.js');
      await new Promise<void>((resolve) => requestAnimationFrame(() => resolve()));
      const pdfOptions = {
        margin: 10,
        filename: `${safeFilename(page.title || 'resume')}.pdf`,
        image: { type: 'jpeg' as const, quality: 0.98 },
        html2canvas: {
          scale: 2,
          useCORS: true,
          backgroundColor: '#ffffff',
          ignoreElements: (element: Element) => element.hasAttribute('data-html2canvas-ignore'),
        },
        jsPDF: { unit: 'mm' as const, format: 'a4' as const, orientation: 'portrait' as const },
        pagebreak: { mode: ['css', 'legacy'], avoid: '.resume-pdf-avoid-break' },
      };
      await html2pdf()
        .set(pdfOptions)
        .from(resumeRef.current)
        .save();
    } catch (e: unknown) {
      setExportError(getErrorMessage(e, t('resume.exportError')));
    } finally {
      resumeRef.current?.classList.remove('resume-pdf-exporting');
      setIsExporting(false);
    }
  };

  const handleRetry = () => {
    setRetryVersion((version) => version + 1);
  };

  return (
    <div className="editorial-container w-full space-y-8">
      <div ref={resumeRef} className="resume-pdf-surface space-y-10 rounded-xl bg-surface-soft p-5 md:p-10 lg:p-12">
        <section className="border-b border-hairline pb-8">
          <div className="flex flex-col gap-5 md:flex-row md:items-end md:justify-between">
            <div>
              <p className="mb-3 text-xs uppercase tracking-[0.28em] text-ochre">
                {t('resume.kicker')}
              </p>
              <h1 className="editorial-page-title">
                {page?.title || t('resume.pageTitleFallback')}
              </h1>
              {page?.subtitle && (
                <p className="mt-4 max-w-2xl text-sm leading-7 tracking-wide text-ink-light opacity-80">
                  {page.subtitle}
                </p>
              )}
            </div>
            {!isLoading && page && hasContent && (
              <Button
                type="button"
                onClick={handleExportPDF}
                loading={isExporting}
                variant="ghost"
                size="md"
                data-html2canvas-ignore="true"
                className="shrink-0"
              >
                {isExporting ? t('resume.exporting') : t('resume.exportPDF')}
              </Button>
            )}
          </div>
        </section>

        {isLoading && (
          <PagePendingState
            variant={page ? 'inline' : 'page'}
            label={t('resume.loading')}
          />
        )}
        <InlineNotice
          message={error}
          action={(
            <Button type="button" variant="ghost" size="sm" onClick={handleRetry}>
              {t('common.retry')}
            </Button>
          )}
        />
        <InlineNotice message={exportError} className="resume-export-notice" />
        {!isLoading && page && !hasContent && (
          <EmptyState illustration="ink-drop" title={t('resume.empty')} />
        )}
        {!isLoading && page && hasContent && (
          <>
            {page.contentMarkdown.trim() && (
              <section className="max-w-3xl">
                <MarkdownRenderer content={page.contentMarkdown} />
              </section>
            )}

            {page.experiences.length > 0 && (
              <TimelineSection title={t('resume.experiences')} items={page.experiences} type="experience" />
            )}

            {page.educations.length > 0 && (
              <TimelineSection title={t('resume.educations')} items={page.educations} type="education" />
            )}

            {Object.keys(skillsByCategory).length > 0 && (
              <section>
                <h2 className="mb-5 text-sm font-bold tracking-[0.24em] text-ink">{t('resume.skills')}</h2>
                <div className="grid gap-5 md:grid-cols-2">
                  {Object.entries(skillsByCategory).map(([category, skills]) => (
                    <div key={category} className="resume-pdf-avoid-break rounded-lg bg-paper p-6">
                      <h3 className="mb-4 font-display text-2xl text-ink">{category}</h3>
                      <div className="space-y-4">
                        {skills.map((skill) => (
                          <SkillBar key={`${skill.category}:${skill.name}`} skill={skill} />
                        ))}
                      </div>
                    </div>
                  ))}
                </div>
              </section>
            )}

            {skillGraph.nodes.length > 1 && (
              <SkillGraphSection
                data={skillGraph}
                projects={projects}
                themeKey={`${effectiveTheme}:${accentColor}`}
                onOpenTag={(tagId) => navigate(`/search?tagId=${tagId}`)}
              />
            )}
          </>
        )}
      </div>
    </div>
  );
};

export default Resume;

type TimelineSectionProps =
  | { title: string; type: 'experience'; items: ResumeExperience[] }
  | { title: string; type: 'education'; items: ResumeEducation[] };

const TimelineSection: React.FC<TimelineSectionProps> = ({ title, items, type }) => (
  <section>
    <h2 className="mb-5 font-display text-3xl text-ink">{title}</h2>
    <div className="space-y-5 border-l border-mountain-grey pl-5">
      {items.map((item, index) => {
        const titleText = type === 'experience'
          ? (item as ResumeExperience).role
          : (item as ResumeEducation).school;
        const metaText = type === 'experience'
          ? [(item as ResumeExperience).organization, item.location].filter(Boolean).join(' / ')
          : [(item as ResumeEducation).degree, (item as ResumeEducation).major, item.location].filter(Boolean).join(' / ');

        return (
          <article key={`${type}:${item.id || index}`} className="resume-pdf-avoid-break relative rounded-lg bg-paper p-6">
            <span className="absolute -left-[1.82rem] top-6 h-3 w-3 rounded-full border border-ochre bg-paper" />
            <div className="flex flex-col gap-3 md:flex-row md:items-start md:justify-between">
              <div>
                <h3 className="font-display text-2xl text-ink">{titleText}</h3>
                <p className="mt-2 text-sm leading-7 tracking-wide text-ink-light">{metaText}</p>
              </div>
              <p className="shrink-0 text-xs tracking-[0.18em] text-ochre">
                {[item.startDate, item.endDate].filter(Boolean).join(' - ')}
              </p>
            </div>
            {item.description && (
              <p className="mt-4 text-sm leading-7 text-ink-light">{item.description}</p>
            )}
            {item.highlights.length > 0 && (
              <ul className="mt-4 space-y-2 text-sm leading-7 text-ink-light">
                {item.highlights.map((highlight) => (
                  <li key={highlight} className="before:mr-2 before:text-ochre before:content-['-']">
                    {highlight}
                  </li>
                ))}
              </ul>
            )}
          </article>
        );
      })}
    </div>
  </section>
);

const SkillBar: React.FC<{ skill: ResumeSkill }> = ({ skill }) => (
  <div>
    <div className="mb-2 flex items-center justify-between gap-3 text-sm">
      <span className="font-bold tracking-wide text-ink">{skill.name}</span>
      <span className="shrink-0 text-xs tracking-[0.16em] text-ink-light">{skill.level}%</span>
    </div>
    <div className="h-2 overflow-hidden bg-[var(--mountain-grey)]">
      <div className="h-full bg-ochre transition-[width]" style={{ width: `${skill.level}%` }} />
    </div>
    {skill.description && <p className="mt-2 text-xs leading-6 text-ink-light">{skill.description}</p>}
  </div>
);

type SkillGraphData = {
  nodes: SkillGraphNode[];
  links: Array<{ source: string; target: string }>;
};

type SkillGraphNode = {
  id: string;
  name: string;
  kind: 'root' | 'category' | 'skill' | 'project' | 'tag';
  tagId?: number;
  relatedProjectIds: string[];
};

const SkillGraphSection: React.FC<{
  data: SkillGraphData;
  projects: ProjectItem[];
  themeKey: string;
  onOpenTag: (tagId: number) => void;
}> = ({ data, projects, themeKey, onOpenTag }) => {
  const language = usePreferenceStore((state) => state.language);
  const t = (key: Parameters<typeof translate>[1]) => translate(language, key);
  const [activeNodeId, setActiveNodeId] = useState(data.nodes[0]?.id || '');
  const activeNode = data.nodes.find((node) => node.id === activeNodeId) || data.nodes[0];
  const relatedProjects = projects.filter((project) => activeNode?.relatedProjectIds.includes(project.id));

  useEffect(() => {
    if (!data.nodes.some((node) => node.id === activeNodeId)) {
      setActiveNodeId(data.nodes[0]?.id || '');
    }
  }, [activeNodeId, data.nodes]);

  return (
    <section data-html2canvas-ignore="true" className="space-y-5">
      <div className="flex flex-col gap-2 md:flex-row md:items-end md:justify-between">
        <div>
          <h2 className="text-sm font-bold tracking-[0.24em] text-ink">{t('resume.skillGraph')}</h2>
          <p className="mt-2 text-sm leading-7 text-ink-light">{t('resume.graphHint')}</p>
        </div>
        {activeNode?.tagId && (
          <button
            type="button"
            onClick={() => onOpenTag(activeNode.tagId as number)}
            className="self-start border border-ochre px-3 py-1.5 text-sm tracking-widest text-ochre transition-colors hover:bg-ochre hover:text-paper md:self-auto"
          >
            {t('resume.openTag')}
          </button>
        )}
      </div>

      <div className="grid gap-5 lg:grid-cols-[minmax(0,1fr)_18rem]">
        <SkillGraph data={data} activeNodeId={activeNodeId} themeKey={themeKey} onSelect={setActiveNodeId} />
        <aside className="border border-mountain-grey bg-[var(--paper-soft)] p-5">
          <p className="text-xs tracking-[0.2em] text-ochre">{activeNode?.name || t('resume.skillGraph')}</p>
          <h3 className="mt-2 text-base font-bold tracking-widest text-ink">{t('resume.relatedProjects')}</h3>
          {relatedProjects.length > 0 ? (
            <div className="mt-4 space-y-3">
              {relatedProjects.map((project) => (
                <article key={project.id} className="border-l border-ochre pl-3">
                  <h4 className="font-bold leading-relaxed text-ink">{project.title}</h4>
                  {project.summary && (
                    <p className="mt-1 line-clamp-3 text-sm leading-6 text-ink-light">{project.summary}</p>
                  )}
                </article>
              ))}
            </div>
          ) : (
            <p className="mt-4 text-sm leading-7 text-ink-light">{t('resume.noRelatedProjects')}</p>
          )}
        </aside>
      </div>
    </section>
  );
};

const SkillGraph: React.FC<{
  data: SkillGraphData;
  activeNodeId: string;
  themeKey: string;
  onSelect: (nodeId: string) => void;
}> = ({ data, activeNodeId, themeKey, onSelect }) => {
  const containerRef = useRef<HTMLDivElement>(null);
  const chartRef = useRef<ECharts | null>(null);
  const onSelectRef = useRef(onSelect);
  onSelectRef.current = onSelect;

  useEffect(() => {
    if (!containerRef.current) {
      return undefined;
    }
    const chart = init(containerRef.current, undefined, { renderer: 'canvas' });
    chartRef.current = chart;
    const resize = () => chart.resize();
    window.addEventListener('resize', resize);
    chart.on('click', (params) => {
      const nodeId = typeof params.data === 'object' && params.data && 'id' in params.data
        ? String((params.data as { id: string }).id)
        : '';
      if (nodeId) {
        onSelectRef.current(nodeId);
      }
    });
    chart.setOption(skillGraphOption(data, activeNodeId, readGraphColors()));

    return () => {
      window.removeEventListener('resize', resize);
      chartRef.current?.dispose();
      chartRef.current = null;
    };
    // init effect 仅在挂载时执行一次；data/activeNodeId/themeKey 变化由下方 setOption effect 处理，
    // onSelect 通过 ref 读取避免闭包过期。
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  useEffect(() => {
    chartRef.current?.setOption(skillGraphOption(data, activeNodeId, readGraphColors()), true);
  }, [activeNodeId, data, themeKey]);

  return (
    <div className="min-h-[24rem] border border-mountain-grey bg-[var(--paper-soft)]">
      <div ref={containerRef} className="h-[24rem] w-full" />
    </div>
  );
};

type GraphColors = {
  ink: string;
  ochre: string;
  mountainGrey: string;
};

const skillGraphOption = (data: SkillGraphData, activeNodeId: string, colors: GraphColors): EChartsCoreOption => ({
  backgroundColor: 'transparent',
  tooltip: { show: true },
  series: [
    {
      type: 'graph',
      layout: 'force',
      roam: true,
      draggable: true,
      animationDuration: 450,
      categories: [
        { name: 'root' },
        { name: 'category' },
        { name: 'skill' },
        { name: 'project' },
        { name: 'tag' },
      ],
      force: {
        repulsion: 180,
        edgeLength: [54, 118],
      },
      label: {
        show: true,
        color: colors.ink,
        fontSize: 12,
        formatter: '{b}',
      },
      edgeSymbol: ['none', 'arrow'],
      edgeSymbolSize: 5,
      lineStyle: {
        color: colors.mountainGrey,
        opacity: 0.78,
      },
      emphasis: {
        focus: 'adjacency',
        lineStyle: { color: colors.ochre, width: 2 },
      },
      data: data.nodes.map((node) => ({
        id: node.id,
        name: node.name,
        category: node.kind,
        symbolSize: nodeSymbolSize(node, activeNodeId),
        itemStyle: {
          color: nodeColor(node.kind, node.id === activeNodeId, colors),
          borderColor: node.id === activeNodeId ? colors.ink : 'transparent',
          borderWidth: node.id === activeNodeId ? 2 : 0,
        },
      })),
      links: data.links,
    },
  ],
});

const nodeSymbolSize = (node: SkillGraphNode, activeNodeId: string) => {
  const base = node.kind === 'root' ? 54 : node.kind === 'category' ? 44 : node.kind === 'project' ? 34 : 30;
  return node.id === activeNodeId ? base + 8 : base;
};

const nodeColor = (kind: SkillGraphNode['kind'], active: boolean, colors: GraphColors) => {
  if (active) {
    return colors.ochre;
  }
  if (kind === 'root') {
    return colors.ink;
  }
  if (kind === 'category') {
    return '#416a8f';
  }
  if (kind === 'project') {
    return '#486f57';
  }
  if (kind === 'tag') {
    return '#7a587d';
  }
  return colors.ochre;
};

const readGraphColors = (): GraphColors => {
  if (typeof window === 'undefined') {
    return { ink: '#1a1a1a', ochre: '#8a3c3a', mountainGrey: '#ebebeb' };
  }
  const styles = window.getComputedStyle(document.documentElement);
  return {
    ink: styles.getPropertyValue('--ink').trim() || '#1a1a1a',
    ochre: styles.getPropertyValue('--ochre').trim() || '#8a3c3a',
    mountainGrey: styles.getPropertyValue('--mountain-grey').trim() || '#ebebeb',
  };
};

const normalizeResumePage = (page: ResumePage): ResumePage => ({
  ...page,
  experiences: page.experiences || [],
  educations: page.educations || [],
  skills: page.skills || [],
});

const groupSkills = (skills: ResumeSkill[]) => skills.reduce<Record<string, ResumeSkill[]>>((groups, skill) => {
  const category = skill.category || 'Skills';
  groups[category] = [...(groups[category] || []), skill];
  return groups;
}, {});

const buildSkillGraph = (skills: ResumeSkill[], projects: ProjectItem[], tags: Tag[]): SkillGraphData => {
  const nodes = new Map<string, SkillGraphNode>();
  const linkSet = new Set<string>();
  const tagByName = new Map(tags.map((tag) => [normalizeToken(tag.name), tag]));
  const projectBySkill = (skillName: string, category: string) => projects
    .filter((project) => projectMatchesSkill(project, skillName, category))
    .map((project) => project.id);

  const addNode = (node: SkillGraphNode) => {
    const existing = nodes.get(node.id);
    if (existing) {
      existing.relatedProjectIds = uniqueStrings([...existing.relatedProjectIds, ...node.relatedProjectIds]);
      return existing;
    }
    nodes.set(node.id, node);
    return node;
  };
  const addLink = (source: string, target: string) => {
    linkSet.add(`${source}=>${target}`);
  };

  addNode({ id: 'root', name: 'Stack', kind: 'root', relatedProjectIds: projects.map((project) => project.id) });

  skills.forEach((skill) => {
    const category = skill.category || 'Skills';
    const categoryId = `category:${category}`;
    const skillId = `skill:${category}:${skill.name}`;
    const relatedProjectIds = projectBySkill(skill.name, category);
    addNode({ id: categoryId, name: category, kind: 'category', relatedProjectIds });
    addNode({ id: skillId, name: skill.name, kind: 'skill', tagId: tagByName.get(normalizeToken(skill.name))?.id, relatedProjectIds });
    addLink('root', categoryId);
    addLink(categoryId, skillId);

    const categoryTag = tagByName.get(normalizeToken(category));
    if (categoryTag) {
      const tagId = `tag:${categoryTag.id}`;
      addNode({ id: tagId, name: `#${categoryTag.name}`, kind: 'tag', tagId: categoryTag.id, relatedProjectIds });
      addLink(categoryId, tagId);
    }
    const skillTag = tagByName.get(normalizeToken(skill.name));
    if (skillTag) {
      const tagId = `tag:${skillTag.id}`;
      addNode({ id: tagId, name: `#${skillTag.name}`, kind: 'tag', tagId: skillTag.id, relatedProjectIds });
      addLink(skillId, tagId);
    }

    projects
      .filter((project) => relatedProjectIds.includes(project.id))
      .forEach((project) => {
        const projectId = `project:${project.id}`;
        addNode({ id: projectId, name: project.title, kind: 'project', relatedProjectIds: [project.id] });
        addLink(skillId, projectId);
      });
  });

  tags.slice(0, 30).forEach((tag) => {
    const relatedProjectIds = projects
      .filter((project) => (project.tags || []).some((name) => normalizeToken(name) === normalizeToken(tag.name)))
      .map((project) => project.id);
    if (relatedProjectIds.length === 0 || nodes.has(`tag:${tag.id}`)) {
      return;
    }
    addNode({ id: `tag:${tag.id}`, name: `#${tag.name}`, kind: 'tag', tagId: tag.id, relatedProjectIds });
    addLink('root', `tag:${tag.id}`);
  });

  return {
    nodes: Array.from(nodes.values()),
    links: Array.from(linkSet).map((value) => {
      const [source, target] = value.split('=>');
      return { source, target };
    }),
  };
};

const projectMatchesSkill = (project: ProjectItem, skillName: string, category: string) => {
  const tokens = [skillName, category].map(normalizeToken).filter(Boolean);
  const tagTokens = (project.tags || []).map(normalizeToken);
  const text = normalizeToken([project.title, project.summary, project.role, project.contentMarkdown].join(' '));
  return tokens.some((token) => tagTokens.includes(token) || text.includes(token));
};

const normalizeToken = (value: string) => value.trim().toLowerCase();

const uniqueStrings = (values: string[]) => Array.from(new Set(values.filter(Boolean)));

const safeFilename = (value: string) => value
  .trim()
  .replace(/[\\/:*?"<>|]+/g, '-')
  .replace(/\s+/g, '-')
  || 'resume';
