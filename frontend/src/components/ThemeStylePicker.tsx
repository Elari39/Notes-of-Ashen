import { useId } from 'react';
import { translate } from '../i18n';
import { usePreferenceStore } from '../store/preferences';
import { themeStyles } from '../store/themeStyles';

const ThemeStylePicker = () => {
  const labelId = useId();
  const language = usePreferenceStore((state) => state.language);
  const themeStyle = usePreferenceStore((state) => state.themeStyle);
  const setThemeStyle = usePreferenceStore((state) => state.setThemeStyle);

  return (
    <div role="group" aria-labelledby={labelId}>
      <p id={labelId} className="mb-2 text-xs tracking-widest text-ink-light">
        {translate(language, 'preferences.styleTitle')}
      </p>
      <div className="theme-style-grid grid grid-cols-2 gap-2">
        {themeStyles.map((style) => (
          <button
            key={style}
            type="button"
            aria-pressed={themeStyle === style}
            aria-label={translate(language, `preferences.style.${style}`)}
            onClick={() => setThemeStyle(style)}
            className="theme-style-option min-h-11 min-w-0 rounded-md border border-hairline p-2 text-left transition-colors hover:border-ink"
          >
            <span className="theme-style-preview" data-preview-style={style} aria-hidden="true">
              <span className="theme-style-preview-heading">Aa<span>札</span></span>
              <span className="theme-style-preview-card"><i /><i /><i /></span>
            </span>
            <span className="mt-2 flex items-center justify-between gap-1 text-xs font-medium text-ink">
              {translate(language, `preferences.style.${style}`)}
              <span aria-hidden="true" className={themeStyle === style ? 'text-ochre' : 'invisible'}>✓</span>
            </span>
            <span className="mt-1 block text-[11px] leading-relaxed text-muted">
              {translate(language, `preferences.styleDescription.${style}`)}
            </span>
          </button>
        ))}
      </div>
    </div>
  );
};

export default ThemeStylePicker;
