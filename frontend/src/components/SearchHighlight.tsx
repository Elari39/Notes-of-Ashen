import React from 'react';

type SearchHighlightProps = {
  value?: string;
  fallback: string;
  className?: string;
};

const SearchHighlight: React.FC<SearchHighlightProps> = ({ value, fallback, className = '' }) => {
  const source = sanitizeHighlight(value || fallback);
  const parts = source.split(/(<mark>|<\/mark>)/i);
  let highlighted = false;

  return (
    <span className={className}>
      {parts.map((part, index) => {
        const lower = part.toLowerCase();
        if (lower === '<mark>') {
          highlighted = true;
          return null;
        }
        if (lower === '</mark>') {
          highlighted = false;
          return null;
        }
        if (!part) {
          return null;
        }
        return highlighted ? (
          <mark key={`${part}:${index}`} className="search-highlight">
            {decodeEntities(part)}
          </mark>
        ) : (
          <React.Fragment key={`${part}:${index}`}>{decodeEntities(part)}</React.Fragment>
        );
      })}
    </span>
  );
};

export default SearchHighlight;

const sanitizeHighlight = (value: string) => value.replace(/<(?!\/?mark\b)[^>]+>/gi, '');

const decodeEntities = (value: string) => {
  if (typeof document === 'undefined') {
    return value;
  }
  const textarea = document.createElement('textarea');
  textarea.innerHTML = value;
  return textarea.value;
};
