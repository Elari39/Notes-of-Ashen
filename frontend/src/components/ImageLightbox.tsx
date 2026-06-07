import React, { useEffect } from 'react';
import { createPortal } from 'react-dom';

export type LightboxImage = {
  src: string;
  alt?: string;
};

type ImageLightboxProps = {
  image: LightboxImage | null;
  onClose: () => void;
};

const ImageLightbox: React.FC<ImageLightboxProps> = ({ image, onClose }) => {
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
      aria-label={image.alt ? `查看图片：${image.alt}` : '查看图片'}
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

export default ImageLightbox;
