import React, { useEffect, useId, useState } from 'react';
import { createCaptcha } from '../api/auth';
import type { CaptchaPurpose } from '../types/api';
import { usePreferenceStore } from '../store/preferences';
import { translate } from '../i18n';
import { getErrorMessage } from '../utils/error';

interface CaptchaFieldProps {
  purpose: CaptchaPurpose;
  captchaId: string;
  captchaCode: string;
  onCaptchaIdChange: (value: string) => void;
  onCaptchaCodeChange: (value: string) => void;
  reloadKey?: number;
}

const CaptchaField: React.FC<CaptchaFieldProps> = ({
  purpose,
  captchaId,
  captchaCode,
  onCaptchaIdChange,
  onCaptchaCodeChange,
  reloadKey = 0,
}) => {
  const language = usePreferenceStore((state) => state.language);
  const [imageData, setImageData] = useState('');
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState('');
  const [manualReloadKey, setManualReloadKey] = useState(0);
  const inputId = useId();
  const errorId = useId();
  const captchaLabel = translate(language, 'auth.captcha');
  const captchaErrorText = translate(language, 'auth.captchaError');
  const captchaLoadingText = translate(language, 'auth.captchaLoading');
  const refreshCaptchaText = translate(language, 'auth.refreshCaptcha');

  useEffect(() => {
    let cancelled = false;
    const loadCaptcha = async () => {
      setLoading(true);
      setError('');
      onCaptchaIdChange('');
      onCaptchaCodeChange('');
      try {
        const res = await createCaptcha({ purpose });
        if (cancelled) {
          return;
        }
        onCaptchaIdChange(res.data.captchaId);
        setImageData(res.data.imageData);
      } catch (err: unknown) {
        if (cancelled) {
          return;
        }
        setImageData('');
        setError(getErrorMessage(err, captchaErrorText));
      } finally {
        if (!cancelled) {
          setLoading(false);
        }
      }
    };

    loadCaptcha();
    return () => {
      cancelled = true;
    };
  }, [captchaErrorText, manualReloadKey, onCaptchaCodeChange, onCaptchaIdChange, purpose, reloadKey]);

  return (
    <div className="space-y-3">
      <div className="flex items-end gap-3">
        <input
          id={inputId}
          type="text"
          inputMode="numeric"
          aria-label={captchaLabel}
          aria-invalid={Boolean(error)}
          aria-describedby={error ? errorId : undefined}
          placeholder={captchaLabel}
          value={captchaCode}
          onChange={(e) => onCaptchaCodeChange(e.target.value.trim())}
          className="min-w-0 flex-1 bg-transparent border-b border-mountain-grey py-2 px-1 text-ink focus:outline-none focus:border-ochre transition-colors placeholder-ink-light placeholder-opacity-50"
          required
        />
        <button
          type="button"
          onClick={() => setManualReloadKey((value) => value + 1)}
          className="h-12 w-32 shrink-0 border border-mountain-grey bg-paper/40 text-xs tracking-widest text-ink-light hover:border-ochre hover:text-ochre transition-colors disabled:opacity-50"
          disabled={loading}
          aria-label={refreshCaptchaText}
          title={refreshCaptchaText}
        >
          {loading ? (
            <span>{captchaLoadingText}</span>
          ) : imageData && captchaId ? (
            <img src={imageData} alt={captchaLabel} className="h-full w-full object-cover" />
          ) : (
            <span>{refreshCaptchaText}</span>
          )}
        </button>
      </div>
      {error && <p id={errorId} role="alert" className="text-xs leading-5 text-danger">{error}</p>}
    </div>
  );
};

export default CaptchaField;
