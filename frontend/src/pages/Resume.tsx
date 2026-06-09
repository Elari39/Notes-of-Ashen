import React, { useEffect, useMemo, useRef, useState } from 'react';
import InlineNotice from '../components/InlineNotice';
import MarkdownRenderer from '../components/MarkdownRenderer';
import { getResumePage } from '../api/siteSettings';
import { usePreferenceStore } from '../store/preferences';
import type { ResumeEducation, ResumeExperience, ResumePage, ResumeSkill } from '../types';
import { getErrorMessage } from '../utils/error';
import { useSEO } from '../utils/seo';

const Resume: React.FC = () => {
  const language = usePreferenceStore((state) => state.language);
  const isZh = language === 'zh';
  const text = getResumeLabels(language);
  const [page, setPage] = useState<ResumePage | null>(null);
  const [isLoading, setIsLoading] = useState(true);
  const [isExporting, setIsExporting] = useState(false);
  const [error, setError] = useState('');
  const [exportError, setExportError] = useState('');
  const resumeRef = useRef<HTMLDivElement>(null);

  useSEO(page?.title || (isZh ? '简介' : 'About'));

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

  const hasContent = Boolean(
    page?.contentMarkdown.trim()
    || page?.experiences.length
    || page?.educations.length
    || page?.skills.length,
  );

  const skillsByCategory = useMemo(() => groupSkills(page?.skills || []), [page?.skills]);

  const handleExportPDF = async () => {
    if (!resumeRef.current || !page || isExporting) {
      return;
    }
    setExportError('');
    setIsExporting(true);
    try {
      const { default: html2pdf } = await import('html2pdf.js');
      await html2pdf()
        .set({
          margin: 10,
          filename: `${safeFilename(page.title || 'resume')}.pdf`,
          image: { type: 'jpeg', quality: 0.98 },
          html2canvas: { scale: 2, useCORS: true },
          jsPDF: { unit: 'mm', format: 'a4', orientation: 'portrait' },
        })
        .from(resumeRef.current)
        .save();
    } catch (e: unknown) {
      setExportError(getErrorMessage(e, text.exportError));
    } finally {
      setIsExporting(false);
    }
  };

  return (
    <div className="mx-auto mt-4 w-full max-w-5xl space-y-8 md:mt-8">
      <div ref={resumeRef} className="space-y-8">
        <section className="border-b border-mountain-grey pb-6">
          <div className="flex flex-col gap-5 md:flex-row md:items-end md:justify-between">
            <div>
              <p className="mb-3 text-xs uppercase tracking-[0.28em] text-ochre">
                {isZh ? 'ABOUT' : 'PROFILE'}
              </p>
              <h1 className="text-3xl font-bold tracking-widest text-ink md:text-4xl">
                {page?.title || (isZh ? '简介' : 'About')}
              </h1>
              {page?.subtitle && (
                <p className="mt-4 max-w-2xl text-sm leading-7 tracking-wide text-ink-light opacity-80">
                  {page.subtitle}
                </p>
              )}
            </div>
            {!isLoading && !error && hasContent && (
              <button
                type="button"
                onClick={handleExportPDF}
                disabled={isExporting}
                className="shrink-0 border border-ochre px-4 py-2 text-sm tracking-widest text-ochre transition-colors hover:bg-ochre hover:text-paper disabled:cursor-not-allowed disabled:opacity-60"
              >
                {isExporting ? text.exporting : text.exportPDF}
              </button>
            )}
          </div>
        </section>

        {isLoading && (
          <p className="py-12 text-center text-sm tracking-[0.24em] text-ink-light">{text.loading}</p>
        )}
        <InlineNotice message={error} />
        <InlineNotice message={exportError} />
        {!isLoading && !error && !hasContent && (
          <section className="border border-mountain-grey bg-[var(--paper-soft)] p-6 text-ink-light">
            <p className="text-base leading-8 tracking-wide">{text.empty}</p>
          </section>
        )}
        {!isLoading && !error && page && hasContent && (
          <>
            {page.contentMarkdown.trim() && (
              <section className="max-w-3xl">
                <MarkdownRenderer content={page.contentMarkdown} />
              </section>
            )}

            {page.experiences.length > 0 && (
              <TimelineSection title={text.experiences} items={page.experiences} type="experience" />
            )}

            {page.educations.length > 0 && (
              <TimelineSection title={text.educations} items={page.educations} type="education" />
            )}

            {Object.keys(skillsByCategory).length > 0 && (
              <section>
                <h2 className="mb-5 text-sm font-bold tracking-[0.24em] text-ink">{text.skills}</h2>
                <div className="grid gap-5 md:grid-cols-2">
                  {Object.entries(skillsByCategory).map(([category, skills]) => (
                    <div key={category} className="border border-mountain-grey bg-[var(--paper-soft)] p-5">
                      <h3 className="mb-4 text-base font-bold tracking-widest text-ink">{category}</h3>
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
    <h2 className="mb-5 text-sm font-bold tracking-[0.24em] text-ink">{title}</h2>
    <div className="space-y-5 border-l border-mountain-grey pl-5">
      {items.map((item, index) => {
        const titleText = type === 'experience'
          ? (item as ResumeExperience).role
          : (item as ResumeEducation).school;
        const metaText = type === 'experience'
          ? [(item as ResumeExperience).organization, item.location].filter(Boolean).join(' / ')
          : [(item as ResumeEducation).degree, (item as ResumeEducation).major, item.location].filter(Boolean).join(' / ');

        return (
          <article key={`${type}:${item.id || index}`} className="relative border border-mountain-grey bg-[var(--paper-soft)] p-5">
            <span className="absolute -left-[1.82rem] top-6 h-3 w-3 rounded-full border border-ochre bg-paper" />
            <div className="flex flex-col gap-3 md:flex-row md:items-start md:justify-between">
              <div>
                <h3 className="text-lg font-bold tracking-widest text-ink">{titleText}</h3>
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

const safeFilename = (value: string) => value
  .trim()
  .replace(/[\\/:*?"<>|]+/g, '-')
  .replace(/\s+/g, '-')
  || 'resume';

const getResumeLabels = (language: string) => language === 'zh'
  ? {
      loading: '简介加载中...',
      loadError: '简介内容加载失败',
      empty: '简介内容还没有填写。',
      exportPDF: '导出 PDF',
      exporting: '导出中...',
      exportError: 'PDF 导出失败',
      experiences: '工作与实习经历',
      educations: '教育背景',
      skills: '技能树',
    }
  : {
      loading: 'Loading profile...',
      loadError: 'Failed to load profile content',
      empty: 'Profile content has not been filled in yet.',
      exportPDF: 'Export PDF',
      exporting: 'Exporting...',
      exportError: 'Failed to export PDF',
      experiences: 'Experience',
      educations: 'Education',
      skills: 'Skills',
    };
