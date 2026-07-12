import React, { useState } from 'react';
import { Link } from 'react-router-dom';
import { resetPassword, sendVerifyCode } from '../api/auth';
import CaptchaField from '../components/CaptchaField';
import InlineNotice from '../components/InlineNotice';
import Button from '../components/ui/Button';
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
    <div className="flex flex-grow items-center justify-center py-4 md:py-10">
      <div className="w-full max-w-xl rounded-xl bg-surface-soft p-5 md:p-10">
        <p className="editorial-kicker mb-4 text-center">{t('auth.recoveryKicker')}</p>
        <h1 className="mb-10 text-center font-display text-5xl leading-tight text-ink">{t('auth.forgotPasswordTitle')}</h1>
        <form onSubmit={handleResetPassword} className="form-panel space-y-6">
          <div>
            <input
              type="email"
              placeholder={t('auth.email')}
              value={email}
              onChange={(e) => setEmail(e.target.value)}
              className="form-control"
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
              className="form-control min-w-0 flex-1"
              required
            />
            <Button
              type="button"
              variant="ghost"
              size="sm"
              onClick={handleSendEmailCode}
              disabled={sendingCode || isResending || !email.trim() || !captchaId || !captchaCode.trim()}
              loading={sendingCode}
              className="h-10 shrink-0"
            >
              {sendingCode
                ? t('auth.sendingEmailCode')
                : isResending
                  ? formatText(t('auth.resendIn'), { n: resendRemaining })
                  : t('auth.sendEmailCode')}
            </Button>
          </div>
          <div>
            <input
              type="password"
              placeholder={t('auth.newPassword')}
              value={newPassword}
              onChange={(e) => setNewPassword(e.target.value)}
              className="form-control"
              required
            />
          </div>
          <div>
            <input
              type="password"
              placeholder={t('auth.confirmPassword')}
              value={confirmPassword}
              onChange={(e) => setConfirmPassword(e.target.value)}
              className="form-control"
              required
            />
          </div>
          <InlineNotice message={error} />
          <InlineNotice message={message} tone="success" />
          <div className="pt-4">
            <Button
              type="submit"
              variant="primary"
              size="lg"
              loading={submitting}
              fullWidth
            >
              {submitting ? t('auth.resetPasswordSubmitting') : t('auth.resetPasswordSubmit')}
            </Button>
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
