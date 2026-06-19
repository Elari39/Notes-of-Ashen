import React, { useEffect, useState } from 'react';
import { AnimatePresence, motion, useReducedMotion } from 'framer-motion';
import { usePreferenceStore } from '../store/preferences';
import { translate } from '../i18n';

const BackToTop: React.FC<{ threshold?: number }> = ({ threshold = 600 }) => {
  const language = usePreferenceStore((state) => state.language);
  const shouldReduceMotion = useReducedMotion();
  const [visible, setVisible] = useState(false);

  useEffect(() => {
    const handleScroll = () => {
      setVisible(window.scrollY > threshold);
    };
    handleScroll();
    window.addEventListener('scroll', handleScroll, { passive: true });
    return () => window.removeEventListener('scroll', handleScroll);
  }, [threshold]);

  const handleClick = () => {
    window.scrollTo({ top: 0, behavior: shouldReduceMotion ? 'auto' : 'smooth' });
  };

  const motionProps = shouldReduceMotion
    ? { initial: { opacity: 1 }, animate: { opacity: 1 }, exit: { opacity: 1 } }
    : { initial: { opacity: 0, scale: 0.9 }, animate: { opacity: 1, scale: 1 }, exit: { opacity: 0, scale: 0.9 } };

  return (
    <AnimatePresence>
      {visible && (
        <motion.button
          type="button"
          onClick={handleClick}
          aria-label={translate(language, 'common.backToTop')}
          transition={{ duration: 0.2, ease: 'easeOut' }}
          className="fixed bottom-6 right-6 z-40 flex h-11 w-11 items-center justify-center border border-mountain-grey bg-paper text-ink shadow-[0_8px_30px_rgba(0,0,0,0.12)] transition-colors hover:border-ochre hover:text-ochre"
          {...motionProps}
        >
          <span className="text-lg leading-none" aria-hidden="true">↑</span>
        </motion.button>
      )}
    </AnimatePresence>
  );
};

export default BackToTop;
