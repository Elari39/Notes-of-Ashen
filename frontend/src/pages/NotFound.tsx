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
        className="relative mx-auto flex w-full max-w-3xl flex-col items-center px-2 py-12 text-center"
      >
        <div className="mb-8 flex w-full max-w-md items-center gap-4 text-xs uppercase tracking-[0.28em] text-ochre">
          <span className="h-px flex-1 bg-ochre opacity-50"></span>
          <span>{t('notFound.kicker')}</span>
          <span className="h-px flex-1 bg-ochre opacity-50"></span>
        </div>

        <h1
          id="not-found-title"
          className="text-7xl font-bold tracking-[0.18em] text-ink opacity-85 sm:text-8xl md:text-9xl"
        >
          404
        </h1>

        <div className="mt-8 h-px w-20 bg-mountain-grey opacity-70"></div>

        <h2 className="mt-8 text-2xl font-bold tracking-widest text-ink sm:text-3xl">
          {t('notFound.title')}
        </h2>
        <p className="mt-6 max-w-xl text-base leading-8 tracking-wide text-ink-light sm:text-lg sm:leading-9">
          {t('notFound.message')}
        </p>

        <Link
          to="/"
          className="mt-10 border border-ink px-7 py-3 text-sm tracking-widest text-ink transition-colors duration-300 hover:bg-ink hover:text-paper focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-4 focus-visible:outline-ochre"
        >
          {t('notFound.back')}
        </Link>
      </section>
    </div>
  );
};

export default NotFound;
