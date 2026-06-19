import React, { useMemo, useState } from 'react';
import { useNavigate, Link } from 'react-router-dom';
import { register, sendVerifyCode } from '../api/auth';
import { useAuthStore } from '../store/auth';
import { usePreferenceStore } from '../store/preferences';
import { useSiteSettingsStore } from '../store/siteSettings';
import { normalizeAvatarUrl } from '../utils/avatar';
import CaptchaField from '../components/CaptchaField';
import FormField from '../components/FormField';
import InlineNotice from '../components/InlineNotice';
import { getErrorMessage } from '../utils/error';
import { formatText, translate } from '../i18n';
import { useFormValidation, type FieldRules } from '../hooks/useFormValidation';
import { useCountdown } from '../hooks/useCountdown';

const ACCOUNT_PATTERN = /^[a-zA-Z0-9_]+$/;
const EMAIL_PATTERN = /^[^\s@]+@[^\s@]+\.[^\s@]+$/;

const Register: React.FC = () => {
  const language = usePreferenceStore((state) => state.language);
  const { registrationEnabled, registrationEmailCodeRequired, hasLoaded, isLoading } = useSiteSettingsStore();
  const [account, setAccount] = useState('');
  const [password, setPassword] = useState('');
  const [confirmPassword, setConfirmPassword] = useState('');
  const [showPassword, setShowPassword] = useState(false);
  const [showConfirmPassword, setShowConfirmPassword] = useState(false);
  const [email, setEmail] = useState('');
  const [emailCode, setEmailCode] = useState('');
  const [nickname, setNickname] = useState('');
  const [avatarUrl, setAvatarUrl] = useState('');
  const [captchaId, setCaptchaId] = useState('');
  const [captchaCode, setCaptchaCode] = useState('');
  const [captchaReloadKey, setCaptchaReloadKey] = useState(0);
  const [error, setError] = useState('');
  const [message, setMessage] = useState('');
  const [submitting, setSubmitting] = useState(false);
  const [sendingCode, setSendingCode] = useState(false);
  const navigate = useNavigate();
  const { setAuth, fetchUser } = useAuthStore();
  const t = (key: Parameters<typeof translate>[1]) => translate(language, key);
  const requiresEmailCode = !hasLoaded || registrationEmailCodeRequired;
  const { remaining: resendRemaining, isCounting: isResending, start: startResendCountdown } = useCountdown(60);

  const rules = useMemo<FieldRules<{ account: string; email: string; password: string; confirmPassword: string }>>(
    () => ({
      account: [
        { type: 'required' },
        { type: 'minLength', value: 3 },
        { type: 'maxLength', value: 64 },
        { type: 'pattern', value: ACCOUNT_PATTERN, key: 'validation.accountPattern' },
      ],
      email: [
        { type: 'required' },
        { type: 'pattern', value: EMAIL_PATTERN, key: 'validation.email' },
      ],
      password: [
        { type: 'required' },
        { type: 'minLength', value: 8 },
      ],
      confirmPassword: [
        { type: 'required' },
        { type: 'match', field: 'password', key: 'validation.passwordMismatch' },
      ],
    }),
    [],
  );

  const { errors, validateField, validate } = useFormValidation(
    { account, email, password, confirmPassword },
    rules,
  );

  const handleRegister = async (e: React.FormEvent) => {
    e.preventDefault();
    if (hasLoaded && !registrationEnabled) {
      setError(t('auth.registrationDisabled'));
      return;
    }
    if (!validate()) {
      return;
    }
    setError('');
    setMessage('');
    setSubmitting(true);
    try {
      const res = await register({
        account: account.trim(),
        password,
        email: email.trim(),
        emailCode: requiresEmailCode ? emailCode.trim() : undefined,
        nickname: nickname.trim(),
        avatarUrl: normalizeAvatarUrl(avatarUrl),
      });
      const token = res.data.accessToken;
      localStorage.setItem('refreshToken', res.data.refreshToken);
      setAuth(null, token);
      await fetchUser();
      navigate('/');
    } catch (err: unknown) {
      setError(getErrorMessage(err, t('auth.registerError')));
      if (requiresEmailCode) {
        setCaptchaReloadKey((value) => value + 1);
      }
    } finally {
      setSubmitting(false);
    }
  };

  const handleSendEmailCode = async () => {
    setError('');
    setMessage('');
    setSendingCode(true);
    try {
      await sendVerifyCode({
        email: email.trim(),
        purpose: 'register',
        captchaId,
        captchaCode,
      });
      setMessage(t('auth.emailCodeSent'));
      startResendCountdown(60);
    } catch (err: unknown) {
      setError(getErrorMessage(err, t('auth.sendEmailCodeError')));
      setCaptchaReloadKey((value) => value + 1);
    } finally {
      setSendingCode(false);
    }
  };

  return (
    <div className="flex-grow flex items-center justify-center">
      <div className="w-full max-w-sm">
        <h1 className="text-3xl font-bold text-ink mb-12 text-center tracking-widest">{t('auth.registerTitle')}</h1>
        {hasLoaded && !registrationEnabled ? (
          <div className="space-y-6 text-center">
            <InlineNotice message={t('auth.registrationDisabled')} />
            <Link to="/login" className="inline-block text-sm tracking-widest text-ochre hover:text-ink transition-colors">
              {t('auth.goLogin')}
            </Link>
          </div>
        ) : (
        <form onSubmit={handleRegister} className="space-y-8">
          <FormField id="register-account" label={t('auth.accountWithHint')} error={errors.account}>
            <input
              type="text"
              placeholder={t('auth.accountWithHint')}
              value={account}
              onChange={(e) => setAccount(e.target.value)}
              onBlur={() => validateField('account')}
              className="w-full bg-transparent border-b border-mountain-grey py-2 px-1 text-ink focus:outline-none focus:border-ochre transition-colors placeholder-ink-light placeholder-opacity-50"
              required
            />
          </FormField>
          <FormField id="register-email" label={t('auth.email')} error={errors.email}>
            <input
              type="email"
              placeholder={t('auth.email')}
              value={email}
              onChange={(e) => setEmail(e.target.value)}
              onBlur={() => validateField('email')}
              className="w-full bg-transparent border-b border-mountain-grey py-2 px-1 text-ink focus:outline-none focus:border-ochre transition-colors placeholder-ink-light placeholder-opacity-50"
              required
            />
          </FormField>
          {requiresEmailCode && (
            <>
              <CaptchaField
                purpose="register"
                captchaId={captchaId}
                captchaCode={captchaCode}
                onCaptchaIdChange={setCaptchaId}
                onCaptchaCodeChange={setCaptchaCode}
                reloadKey={captchaReloadKey}
              />
              <div className="flex items-end gap-3">
                <input
                  type="text"
                  inputMode="numeric"
                  placeholder={t('auth.emailCode')}
                  value={emailCode}
                  onChange={(e) => setEmailCode(e.target.value.trim())}
                  className="min-w-0 flex-1 bg-transparent border-b border-mountain-grey py-2 px-1 text-ink focus:outline-none focus:border-ochre transition-colors placeholder-ink-light placeholder-opacity-50"
                  required
                />
                <button
                  type="button"
                  onClick={handleSendEmailCode}
                  disabled={sendingCode || isResending || !email.trim() || !captchaId || !captchaCode.trim()}
                  className="h-10 shrink-0 border border-ink px-4 text-xs tracking-widest text-ink hover:bg-ink hover:text-paper transition-colors disabled:opacity-50 disabled:cursor-not-allowed"
                >
                  {sendingCode
                    ? t('auth.sendingEmailCode')
                    : isResending
                      ? formatText(t('auth.resendIn'), { n: resendRemaining })
                      : t('auth.sendEmailCode')}
                </button>
              </div>
            </>
          )}
          <div>
            <input
              type="text"
              placeholder={t('auth.nicknameOptional')}
              value={nickname}
              onChange={(e) => setNickname(e.target.value)}
              className="w-full bg-transparent border-b border-mountain-grey py-2 px-1 text-ink focus:outline-none focus:border-ochre transition-colors placeholder-ink-light placeholder-opacity-50"
            />
          </div>
          <div>
            <input
              type="text"
              placeholder={t('auth.avatarOptional')}
              value={avatarUrl}
              onChange={(e) => setAvatarUrl(e.target.value)}
              className="w-full bg-transparent border-b border-mountain-grey py-2 px-1 text-ink focus:outline-none focus:border-ochre transition-colors placeholder-ink-light placeholder-opacity-50"
            />
          </div>
          <div>
            <label htmlFor="register-password" className="mb-2 block text-xs tracking-widest text-ink-light">
              {t('auth.passwordWithHint')}
            </label>
            <div className="relative">
              <input
                id="register-password"
                type={showPassword ? 'text' : 'password'}
                placeholder={t('auth.passwordWithHint')}
                value={password}
                onChange={(e) => setPassword(e.target.value)}
                onBlur={() => validateField('password')}
                aria-invalid={errors.password ? true : undefined}
                aria-describedby={errors.password ? 'register-password-error' : undefined}
                className="w-full bg-transparent border-b border-mountain-grey py-2 pl-1 pr-16 text-ink focus:outline-none focus:border-ochre transition-colors placeholder-ink-light placeholder-opacity-50"
                required
              />
              <button
                type="button"
                onClick={() => setShowPassword((visible) => !visible)}
                className="absolute right-1 top-1/2 -translate-y-1/2 text-xs tracking-widest text-ink-light hover:text-ochre transition-colors"
                aria-label={showPassword ? t('auth.hidePassword') : t('auth.showPassword')}
                title={showPassword ? t('auth.hidePassword') : t('auth.showPassword')}
              >
                {showPassword ? t('auth.hidePassword') : t('auth.showPassword')}
              </button>
            </div>
            {errors.password && (
              <p id="register-password-error" role="alert" className="mt-2 border-l-2 border-ochre bg-[var(--notice-bg)] px-3 py-2 text-xs text-ochre">
                {errors.password}
              </p>
            )}
          </div>
          <div>
            <label htmlFor="register-confirm-password" className="mb-2 block text-xs tracking-widest text-ink-light">
              {t('auth.confirmPasswordWithHint')}
            </label>
            <div className="relative">
              <input
                id="register-confirm-password"
                type={showConfirmPassword ? 'text' : 'password'}
                placeholder={t('auth.confirmPasswordWithHint')}
                value={confirmPassword}
                onChange={(e) => setConfirmPassword(e.target.value)}
                onBlur={() => validateField('confirmPassword')}
                aria-invalid={errors.confirmPassword ? true : undefined}
                aria-describedby={errors.confirmPassword ? 'register-confirm-password-error' : undefined}
                className="w-full bg-transparent border-b border-mountain-grey py-2 pl-1 pr-16 text-ink focus:outline-none focus:border-ochre transition-colors placeholder-ink-light placeholder-opacity-50"
                required
              />
              <button
                type="button"
                onClick={() => setShowConfirmPassword((visible) => !visible)}
                className="absolute right-1 top-1/2 -translate-y-1/2 text-xs tracking-widest text-ink-light hover:text-ochre transition-colors"
                aria-label={showConfirmPassword ? t('auth.hidePassword') : t('auth.showPassword')}
                title={showConfirmPassword ? t('auth.hidePassword') : t('auth.showPassword')}
              >
                {showConfirmPassword ? t('auth.hidePassword') : t('auth.showPassword')}
              </button>
            </div>
            {errors.confirmPassword && (
              <p id="register-confirm-password-error" role="alert" className="mt-2 border-l-2 border-ochre bg-[var(--notice-bg)] px-3 py-2 text-xs text-ochre">
                {errors.confirmPassword}
              </p>
            )}
          </div>
          <InlineNotice message={error} />
          <InlineNotice message={message} tone="success" />
          <div className="pt-4">
            <button
              type="submit"
              disabled={submitting || isLoading}
              className="w-full border border-ink text-ink py-3 hover:bg-ink hover:text-paper transition-colors duration-300 tracking-widest disabled:opacity-50 disabled:cursor-not-allowed"
            >
              {submitting || isLoading ? t('auth.registerSubmitting') : t('auth.registerSubmit')}
            </button>
          </div>
          <div className="text-center text-sm text-ink-light opacity-70 mt-4">
            {t('auth.hasAccount')} <Link to="/login" className="hover:text-ochre transition-colors">{t('auth.goLogin')}</Link>
          </div>
        </form>
        )}
      </div>
    </div>
  );
};

export default Register;
