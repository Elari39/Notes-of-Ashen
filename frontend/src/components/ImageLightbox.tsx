import React, { useEffect, useRef } from 'react';
import { createPortal } from 'react-dom';
import { usePreferenceStore } from '../store/preferences';
import { formatText, translate } from '../i18n';
import { trapFocus } from '../utils/focusTrap';

export type LightboxImage = {
  src: string;
  alt?: string;
};

type ImageLightboxProps = {
  image: LightboxImage | null;
  onClose: () => void;
};

const ImageLightbox: React.FC<ImageLightboxProps> = ({ image, onClose }) => {
  const language = usePreferenceStore((state) => state.language);
  const t = (key: Parameters<typeof translate>[1]) => translate(language, key);
  const containerRef = useRef<HTMLDivElement | null>(null);

  useEffect(() => {
    if (!image || typeof document === 'undefined') {
      return undefined;
    }

    const previouslyFocused = document.activeElement as HTMLElement | null;

    const handleKeyDown = (event: KeyboardEvent) => {
      if (event.key === 'Escape') {
        onClose();
        return;
      }
      if (containerRef.current) {
        trapFocus(containerRef.current, event);
      }
    };

    const previousOverflow = document.body.style.overflow;
    document.body.style.overflow = 'hidden';
    document.addEventListener('keydown', handleKeyDown);

    // 打开后将焦点移入对话框，便于键盘用户操作关闭按钮。
    const focusTimer = window.setTimeout(() => {
      const focusable = containerRef.current?.querySelectorAll<HTMLElement>('[data-lightbox-focus]');
      focusable?.[0]?.focus();
    }, 0);

    return () => {
      window.clearTimeout(focusTimer);
      document.body.style.overflow = previousOverflow;
      document.removeEventListener('keydown', handleKeyDown);
      // 关闭后恢复焦点到触发元素。
      if (previouslyFocused && typeof previouslyFocused.focus === 'function') {
        previouslyFocused.focus();
      }
    };
  }, [image, onClose]);

  if (!image || typeof document === 'undefined') {
    return null;
  }

  return createPortal(
    <div
      ref={containerRef}
      className="image-lightbox"
      role="dialog"
      aria-modal="true"
      aria-label={image.alt ? formatText(t('imageLightbox.viewImageWithAlt'), { alt: image.alt }) : t('imageLightbox.viewImage')}
      onClick={onClose}
    >
      <img
        src={image.src}
        alt={image.alt || ''}
        className="image-lightbox__image"
        onClick={event => event.stopPropagation()}
      />
      <button
        type="button"
        data-lightbox-focus
        onClick={onClose}
        aria-label={t('imageLightbox.close')}
        className="absolute right-4 top-4 inline-flex h-10 w-10 items-center justify-center rounded-full border border-white/60 bg-black/40 text-white hover:bg-black/60 focus:outline-hidden focus:ring-2 focus:ring-ochre"
      >
        <span aria-hidden="true" className="text-xl leading-none">&times;</span>
      </button>
    </div>,
    document.body,
  );
};

export default ImageLightbox;
