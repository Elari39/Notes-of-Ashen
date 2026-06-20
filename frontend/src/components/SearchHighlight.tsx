import React, { memo } from 'react';

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

export default memo(SearchHighlight);

const sanitizeHighlight = (value: string) => value.replace(/<(?!\/?mark\b)[^>]+>/gi, '');

// 纯函数解码常见 HTML 实体，避免共享 textarea 节点的重入与潜在 XSS 隐患。
const entityMap: ReadonlyArray<[RegExp, string]> = [
  [/&lt;/gi, '<'],
  [/&gt;/gi, '>'],
  [/&quot;/gi, '"'],
  [/&#39;/gi, "'"],
  [/&nbsp;/gi, ' '],
  [/&amp;/gi, '&'],
];

const decodeEntities = (value: string) => {
  let result = value;
  for (const [pattern, replacement] of entityMap) {
    result = result.replace(pattern, replacement);
  }
  return result;
};
