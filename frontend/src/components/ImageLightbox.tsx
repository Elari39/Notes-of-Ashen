import React, { useEffect } from 'react';
import { createPortal } from 'react-dom';
import { usePreferenceStore } from '../store/preferences';

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

  useEffect(() => {
    if (!image || typeof document === 'undefined') {
      return undefined;
    }

    const handleKeyDown = (event: KeyboardEvent) => {
      if (event.key === 'Escape') {
        onClose();
      }
    };

    const previousOverflow = document.body.style.overflow;
    document.body.style.overflow = 'hidden';
    document.addEventListener('keydown', handleKeyDown);

    return () => {
      document.body.style.overflow = previousOverflow;
      document.removeEventListener('keydown', handleKeyDown);
    };
  }, [image, onClose]);

  if (!image || typeof document === 'undefined') {
    return null;
  }

  return createPortal(
    <div
      className="image-lightbox"
      role="dialog"
      aria-modal="true"
      aria-label={image.alt ? imageLightboxLabel(language, image.alt) : imageLightboxLabel(language)}
      onClick={onClose}
    >
      <img
        src={image.src}
        alt={image.alt || ''}
        className="image-lightbox__image"
        onClick={event => event.stopPropagation()}
      />
    </div>,
    document.body,
  );
};

const imageLightboxLabel = (language: string, alt = '') => {
  if (language === 'zh') {
    return alt ? `查看图片：${alt}` : '查看图片';
  }
  return alt ? `View image: ${alt}` : 'View image';
};

export default ImageLightbox;
