import React, { useEffect, useState } from 'react';
import InlineNotice from '../components/InlineNotice';
import MarkdownRenderer from '../components/MarkdownRenderer';
import { getResumePage } from '../api/siteSettings';
import { usePreferenceStore } from '../store/preferences';
import type { ResumePage } from '../types';
import { getErrorMessage } from '../utils/error';
import { useSEO } from '../utils/seo';

const Resume: React.FC = () => {
  const language = usePreferenceStore((state) => state.language);
  const isZh = language === 'zh';
  const text = getResumeLabels(language);
  const [page, setPage] = useState<ResumePage | null>(null);
  const [isLoading, setIsLoading] = useState(true);
  const [error, setError] = useState('');

  useSEO(page?.title || (isZh ? '简介' : 'About'));

  useEffect(() => {
    let active = true;
    setIsLoading(true);
    setError('');
    getResumePage()
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
    <div className="mx-auto mt-4 w-full max-w-3xl space-y-8 md:mt-8">
      <section className="border-b border-mountain-grey pb-6">
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
      </section>

      {isLoading && (
        <p className="py-12 text-center text-sm tracking-[0.24em] text-ink-light">{text.loading}</p>
      )}
      <InlineNotice message={error} />
      {!isLoading && !error && !page?.contentMarkdown.trim() && (
        <section className="border border-mountain-grey bg-[var(--paper-soft)] p-6 text-ink-light">
          <p className="text-base leading-8 tracking-wide">{text.empty}</p>
        </section>
      )}
      {!isLoading && !error && page?.contentMarkdown.trim() && (
        <MarkdownRenderer content={page.contentMarkdown} />
      )}
    </div>
  );
};

export default Resume;

const getResumeLabels = (language: string) => language === 'zh'
  ? {
      loading: '简介加载中...',
      loadError: '简介内容加载失败',
      empty: '简介内容还没有填写。',
    }
  : {
      loading: 'Loading profile...',
      loadError: 'Failed to load profile content',
      empty: 'Profile content has not been filled in yet.',
    };
