import React from 'react';
import { Link } from 'react-router-dom';
import { translate } from '../i18n';
import { usePreferenceStore } from '../store/preferences';

const NotFound: React.FC = () => {
  const language = usePreferenceStore((state) => state.language);
  const t = (key: Parameters<typeof translate>[1]) => translate(language, key);

  return (
    <div className="flex-grow flex items-center justify-center">
      <section
        aria-labelledby="not-found-title"
        className="relative mx-auto flex w-full max-w-4xl flex-col items-center overflow-hidden rounded-xl bg-surface-dark px-6 py-16 text-center text-on-dark md:py-24"
      >
        <div className="mb-8 flex w-full max-w-md items-center gap-4 text-xs uppercase tracking-[0.28em] text-ochre">
          <span className="h-px flex-1 bg-ochre opacity-50"></span>
          <span>{t('notFound.kicker')}</span>
          <span className="h-px flex-1 bg-ochre opacity-50"></span>
        </div>

        <h1
          id="not-found-title"
          className="font-display text-8xl tracking-[-0.04em] text-on-dark sm:text-9xl md:text-[10rem]"
        >
          404
        </h1>

        <div className="mt-8 h-px w-20 bg-mountain-grey opacity-70"></div>

        <h2 className="mt-8 font-display text-3xl text-on-dark sm:text-4xl">
          {t('notFound.title')}
        </h2>
        <p className="mt-6 max-w-xl text-base leading-8 text-on-dark-soft sm:text-lg sm:leading-9">
          {t('notFound.message')}
        </p>

        <Link
          to="/"
          className="mt-10 inline-flex min-h-11 items-center rounded-md bg-ochre px-7 py-3 text-sm font-medium text-on-accent transition-[filter] hover:brightness-95 focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-4 focus-visible:outline-ochre"
        >
          {t('notFound.back')}
        </Link>
      </section>
    </div>
  );
};

export default NotFound;
