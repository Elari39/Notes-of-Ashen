import React from 'react';
import { Link } from 'react-router-dom';
import { translate } from '../i18n';
import { usePreferenceStore } from '../store/preferences';

const NotFound: React.FC = () => {
  const language = usePreferenceStore((state) => state.language);
  const t = (key: Parameters<typeof translate>[1]) => translate(language, key);

  return (
    <div className="flex-grow flex flex-col items-center justify-center space-y-8 text-center">
      <h1 className="text-6xl md:text-8xl font-bold text-ink tracking-widest opacity-80">404</h1>
      <p className="text-xl text-ink-light tracking-widest italic">{t('notFound.message')}</p>
      <div className="w-16 h-px bg-mountain-grey opacity-50"></div>
      <Link to="/" className="px-6 py-2 border border-ink text-ink hover:bg-ink hover:text-paper transition-colors duration-300 tracking-widest">
        {t('notFound.back')}
      </Link>
    </div>
  );
};

export default NotFound;
