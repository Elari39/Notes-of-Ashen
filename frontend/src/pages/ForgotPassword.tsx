import React, { useState } from 'react';
import { Link } from 'react-router-dom';
import { resetPassword, sendVerifyCode } from '../api/auth';
import CaptchaField from '../components/CaptchaField';
import InlineNotice from '../components/InlineNotice';
import { useCountdown } from '../hooks/useCountdown';
import { formatText, translate } from '../i18n';
import { usePreferenceStore } from '../store/preferences';
import { getErrorMessage } from '../utils/error';

const ForgotPassword: React.FC = () => {
  const language = usePreferenceStore((state) => state.language);
  const [email, setEmail] = useState('');
  const [emailCode, setEmailCode] = useState('');
  const [newPassword, setNewPassword] = useState('');
  const [confirmPassword, setConfirmPassword] = useState('');
  const [captchaId, setCaptchaId] = useState('');
  const [captchaCode, setCaptchaCode] = useState('');
  const [captchaReloadKey, setCaptchaReloadKey] = useState(0);
  const [error, setError] = useState('');
  const [message, setMessage] = useState('');
  const [sendingCode, setSendingCode] = useState(false);
  const [submitting, setSubmitting] = useState(false);
  const t = (key: Parameters<typeof translate>[1]) => translate(language, key);
  const { remaining: resendRemaining, isCounting: isResending, start: startResendCountdown } = useCountdown(60);

  const handleSendEmailCode = async () => {
    setError('');
    setMessage('');
    setSendingCode(true);
    try {
      await sendVerifyCode({
        email: email.trim(),
        purpose: 'reset_password',
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

  const handleResetPassword = async (e: React.FormEvent) => {
    e.preventDefault();
    if (newPassword !== confirmPassword) {
      setError(t('auth.passwordMismatch'));
      return;
    }
    setError('');
    setMessage('');
    setSubmitting(true);
    try {
      await resetPassword({
        email: email.trim(),
        emailCode: emailCode.trim(),
        newPassword,
      });
      setNewPassword('');
      setConfirmPassword('');
      setEmailCode('');
      setMessage(t('auth.resetPasswordSuccess'));
    } catch (err: unknown) {
      setError(getErrorMessage(err, t('auth.resetPasswordError')));
    } finally {
      setSubmitting(false);
    }
  };

  return (
    <div className="flex-grow flex items-center justify-center">
      <div className="w-full max-w-sm">
        <h1 className="text-3xl font-bold text-ink mb-12 text-center tracking-widest">{t('auth.forgotPasswordTitle')}</h1>
        <form onSubmit={handleResetPassword} className="space-y-8">
          <div>
            <input
              type="email"
              placeholder={t('auth.email')}
              value={email}
              onChange={(e) => setEmail(e.target.value)}
              className="w-full bg-transparent border-b border-mountain-grey py-2 px-1 text-ink focus:outline-none focus:border-ochre transition-colors placeholder-ink-light placeholder-opacity-50"
              required
            />
          </div>
          <CaptchaField
            purpose="reset_password"
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
          <div>
            <input
              type="password"
              placeholder={t('auth.newPassword')}
              value={newPassword}
              onChange={(e) => setNewPassword(e.target.value)}
              className="w-full bg-transparent border-b border-mountain-grey py-2 px-1 text-ink focus:outline-none focus:border-ochre transition-colors placeholder-ink-light placeholder-opacity-50"
              required
            />
          </div>
          <div>
            <input
              type="password"
              placeholder={t('auth.confirmPassword')}
              value={confirmPassword}
              onChange={(e) => setConfirmPassword(e.target.value)}
              className="w-full bg-transparent border-b border-mountain-grey py-2 px-1 text-ink focus:outline-none focus:border-ochre transition-colors placeholder-ink-light placeholder-opacity-50"
              required
            />
          </div>
          <InlineNotice message={error} />
          <InlineNotice message={message} tone="success" />
          <div className="pt-4">
            <button
              type="submit"
              disabled={submitting}
              className="w-full border border-ink text-ink py-3 hover:bg-ink hover:text-paper transition-colors duration-300 tracking-widest disabled:opacity-50 disabled:cursor-not-allowed"
            >
              {submitting ? t('auth.resetPasswordSubmitting') : t('auth.resetPasswordSubmit')}
            </button>
          </div>
          <div className="text-center text-sm text-ink-light opacity-70 mt-4">
            <Link to="/login" className="hover:text-ochre transition-colors">{t('auth.goLogin')}</Link>
          </div>
        </form>
      </div>
    </div>
  );
};

export default ForgotPassword;
