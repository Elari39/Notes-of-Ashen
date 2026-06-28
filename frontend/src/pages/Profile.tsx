import React, { useEffect, useMemo, useState } from 'react';
import { useAuthStore } from '../store/auth';
import { usePreferenceStore } from '../store/preferences';
import { sendCurrentUserVerifyCode, updateCurrentUser, updatePassword } from '../api/user';
import { isHttpAvatarUrl, normalizeAvatarUrl } from '../utils/avatar';
import CaptchaField from '../components/CaptchaField';
import InlineNotice from '../components/InlineNotice';
import Button from '../components/ui/Button';
import { getErrorMessage } from '../utils/error';
import { translate } from '../i18n';
import { useFormValidation, type FieldRules } from '../hooks/useFormValidation';
import { useShallow } from 'zustand/react/shallow';

const Profile: React.FC = () => {
  const { user, fetchUser } = useAuthStore(
    useShallow((state) => ({ user: state.user, fetchUser: state.fetchUser })),
  );
  const language = usePreferenceStore((state) => state.language);
  const [nickname, setNickname] = useState(user?.nickname || '');
  const [email, setEmail] = useState(user?.email || '');
  const [profileEmailCode, setProfileEmailCode] = useState('');
  const [profileCaptchaId, setProfileCaptchaId] = useState('');
  const [profileCaptchaCode, setProfileCaptchaCode] = useState('');
  const [profileCaptchaReloadKey, setProfileCaptchaReloadKey] = useState(0);
  const [avatarUrl, setAvatarUrl] = useState(user?.avatarUrl || '');

  const [oldPassword, setOldPassword] = useState('');
  const [newPassword, setNewPassword] = useState('');
  const [confirmPassword, setConfirmPassword] = useState('');
  const [passwordEmailCode, setPasswordEmailCode] = useState('');
  const [passwordCaptchaId, setPasswordCaptchaId] = useState('');
  const [passwordCaptchaCode, setPasswordCaptchaCode] = useState('');
  const [passwordCaptchaReloadKey, setPasswordCaptchaReloadKey] = useState(0);
  const [pwdError, setPwdError] = useState('');
  const [pwdMsg, setPwdMsg] = useState('');
  const [msg, setMsg] = useState('');
  const [profileUpdated, setProfileUpdated] = useState(false);
  const [profileSubmitting, setProfileSubmitting] = useState(false);
  const [passwordSubmitting, setPasswordSubmitting] = useState(false);
  const [profileCodeSending, setProfileCodeSending] = useState(false);
  const [passwordCodeSending, setPasswordCodeSending] = useState(false);
  const previewAvatarUrl = normalizeAvatarUrl(avatarUrl);
  const shouldShowAvatarPreview = isHttpAvatarUrl(previewAvatarUrl);
  const profileEmailChanged = email.trim().toLowerCase() !== (user?.email || '').trim().toLowerCase();
  const t = (key: Parameters<typeof translate>[1]) => translate(language, key);

  const passwordRules = useMemo<FieldRules<{ oldPassword: string; newPassword: string; confirmPassword: string }>>(
    () => ({
      oldPassword: [{ type: 'required' }],
      newPassword: [{ type: 'required' }, { type: 'minLength', value: 8 }],
      confirmPassword: [{ type: 'required' }, { type: 'match', field: 'newPassword' }],
    }),
    [],
  );
  const { errors: pwdFieldErrors, validate: validatePassword } = useFormValidation(
    { oldPassword, newPassword, confirmPassword },
    passwordRules,
  );

  useEffect(() => {
    setNickname(user?.nickname || '');
    setEmail(user?.email || '');
    setProfileEmailCode('');
    setAvatarUrl(user?.avatarUrl || '');
  }, [user]);

  const handleUpdateProfile = async (e: React.FormEvent) => {
    e.preventDefault();
    setMsg('');
    setProfileUpdated(false);
    setProfileSubmitting(true);
    try {
      await updateCurrentUser({
        nickname: nickname.trim(),
        email: email.trim(),
        emailCode: profileEmailChanged ? profileEmailCode.trim() : undefined,
        avatarUrl: normalizeAvatarUrl(avatarUrl),
      });
      await fetchUser();
      setMsg(t('profile.updated'));
      setProfileUpdated(true);
      setProfileEmailCode('');
    } catch (err: unknown) {
      setMsg(getErrorMessage(err, t('profile.updateError')));
    } finally {
      setProfileSubmitting(false);
    }
  };

  const handleSendProfileCode = async () => {
    setMsg('');
    setProfileUpdated(false);
    setProfileCodeSending(true);
    try {
      await sendCurrentUserVerifyCode({
        email: email.trim(),
        purpose: 'update_email',
        captchaId: profileCaptchaId,
        captchaCode: profileCaptchaCode,
      });
      setMsg(t('auth.emailCodeSent'));
      setProfileUpdated(true);
    } catch (err: unknown) {
      setMsg(getErrorMessage(err, t('auth.sendEmailCodeError')));
      setProfileCaptchaReloadKey((value) => value + 1);
    } finally {
      setProfileCodeSending(false);
    }
  };

  const handleSendPasswordCode = async () => {
    setPwdError('');
    setPwdMsg('');
    setPasswordCodeSending(true);
    try {
      await sendCurrentUserVerifyCode({
        purpose: 'change_password',
        captchaId: passwordCaptchaId,
        captchaCode: passwordCaptchaCode,
      });
      setPwdMsg(t('auth.emailCodeSent'));
    } catch (err: unknown) {
      setPwdError(getErrorMessage(err, t('auth.sendEmailCodeError')));
      setPasswordCaptchaReloadKey((value) => value + 1);
    } finally {
      setPasswordCodeSending(false);
    }
  };

  const handleUpdatePassword = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!validatePassword()) return;
    setPwdError('');
    setPwdMsg('');
    setPasswordSubmitting(true);
    try {
      await updatePassword({ oldPassword, newPassword, emailCode: passwordEmailCode.trim() });
      setOldPassword('');
      setNewPassword('');
      setConfirmPassword('');
      setPasswordEmailCode('');
      setPwdMsg(t('profile.passwordUpdated'));
    } catch (err: unknown) {
      setPwdError(getErrorMessage(err, t('profile.updateError')));
    } finally {
      setPasswordSubmitting(false);
    }
  };

  return (
    <div className="max-w-md mx-auto w-full space-y-16">
      <section>
        <h2 className="text-2xl font-bold text-ink mb-8 tracking-widest text-center">{t('profile.title')}</h2>
        <form onSubmit={handleUpdateProfile} className="space-y-6">
          <div>
            <label className="block text-sm text-ink-light opacity-70 mb-2">{t('profile.accountLabel')}</label>
            <input
              type="text"
              value={user?.account ?? ''}
              disabled
              className="w-full bg-transparent border-b border-mountain-grey py-2 px-1 text-ink-light opacity-50 cursor-not-allowed"
            />
          </div>
          <div>
            <label className="block text-sm text-ink-light opacity-70 mb-2">{t('profile.nicknameLabel')}</label>
            <input
              type="text"
              value={nickname}
              onChange={e => setNickname(e.target.value)}
              className="w-full bg-transparent border-b border-mountain-grey py-2 px-1 text-ink focus:outline-none focus:border-ochre transition-colors"
            />
          </div>
          <div>
            <label className="block text-sm text-ink-light opacity-70 mb-2">{t('profile.emailLabel')}</label>
            <input
              type="email"
              value={email}
              onChange={e => setEmail(e.target.value)}
              className="w-full bg-transparent border-b border-mountain-grey py-2 px-1 text-ink focus:outline-none focus:border-ochre transition-colors"
            />
          </div>
          {profileEmailChanged && (
            <>
              <CaptchaField
                purpose="update_email"
                captchaId={profileCaptchaId}
                captchaCode={profileCaptchaCode}
                onCaptchaIdChange={setProfileCaptchaId}
                onCaptchaCodeChange={setProfileCaptchaCode}
                reloadKey={profileCaptchaReloadKey}
              />
              <div className="flex items-end gap-3">
                <input
                  type="text"
                  inputMode="numeric"
                  placeholder={t('auth.emailCode')}
                  value={profileEmailCode}
                  onChange={(e) => setProfileEmailCode(e.target.value.trim())}
                  className="min-w-0 flex-1 bg-transparent border-b border-mountain-grey py-2 px-1 text-ink focus:outline-none focus:border-ochre transition-colors placeholder-ink-light placeholder-opacity-50"
                  required
                />
                <Button
                  type="button"
                  variant="ghost"
                  size="sm"
                  onClick={handleSendProfileCode}
                  disabled={profileCodeSending || !email.trim() || !profileCaptchaId || !profileCaptchaCode.trim()}
                  loading={profileCodeSending}
                  className="h-10 shrink-0"
                >
                  {profileCodeSending ? t('auth.sendingEmailCode') : t('auth.sendEmailCode')}
                </Button>
              </div>
            </>
          )}
          <div>
            <label className="block text-sm text-ink-light opacity-70 mb-2">{t('profile.avatarLabel')}</label>
            {shouldShowAvatarPreview && (
              <img
                src={previewAvatarUrl}
                alt="avatar"
                className="mb-3 w-10 h-10 rounded-full border border-mountain-grey object-cover"
              />
            )}
            <input
              type="text"
              value={avatarUrl}
              onChange={e => setAvatarUrl(e.target.value)}
              className="w-full bg-transparent border-b border-mountain-grey py-2 px-1 text-ink focus:outline-none focus:border-ochre transition-colors"
            />
          </div>
          <InlineNotice message={msg} tone={profileUpdated ? 'success' : 'error'} />
          <div className="pt-4 text-center">
            <Button type="submit" variant="primary" size="md" loading={profileSubmitting}>
              {profileSubmitting ? t('profile.updatingProfile') : t('profile.updateProfile')}
            </Button>
          </div>
        </form>
      </section>

      <div className="w-24 h-px bg-mountain-grey mx-auto opacity-50"></div>

      <section>
        <h2 className="text-2xl font-bold text-ink mb-8 tracking-widest text-center">{t('profile.passwordTitle')}</h2>
        <form onSubmit={handleUpdatePassword} className="space-y-6">
          <div>
            <input
              type="password"
              placeholder={t('auth.oldPassword')}
              value={oldPassword}
              onChange={e => setOldPassword(e.target.value)}
              required
              className="w-full bg-transparent border-b border-mountain-grey py-2 px-1 text-ink focus:outline-none focus:border-ochre transition-colors placeholder-ink-light placeholder-opacity-50"
            />
            {pwdFieldErrors.oldPassword && (
              <p className="mt-1 text-xs text-ember">{pwdFieldErrors.oldPassword}</p>
            )}
          </div>
          <div>
            <input
              type="password"
              placeholder={t('auth.newPassword')}
              value={newPassword}
              onChange={e => setNewPassword(e.target.value)}
              required
              className="w-full bg-transparent border-b border-mountain-grey py-2 px-1 text-ink focus:outline-none focus:border-ochre transition-colors placeholder-ink-light placeholder-opacity-50"
            />
            {pwdFieldErrors.newPassword && (
              <p className="mt-1 text-xs text-ember">{pwdFieldErrors.newPassword}</p>
            )}
          </div>
          <div>
            <input
              type="password"
              placeholder={t('auth.confirmPassword')}
              value={confirmPassword}
              onChange={e => setConfirmPassword(e.target.value)}
              required
              className="w-full bg-transparent border-b border-mountain-grey py-2 px-1 text-ink focus:outline-none focus:border-ochre transition-colors placeholder-ink-light placeholder-opacity-50"
            />
            {pwdFieldErrors.confirmPassword && (
              <p className="mt-1 text-xs text-ember">{pwdFieldErrors.confirmPassword}</p>
            )}
          </div>
          <CaptchaField
            purpose="change_password"
            captchaId={passwordCaptchaId}
            captchaCode={passwordCaptchaCode}
            onCaptchaIdChange={setPasswordCaptchaId}
            onCaptchaCodeChange={setPasswordCaptchaCode}
            reloadKey={passwordCaptchaReloadKey}
          />
          <div className="flex items-end gap-3">
            <input
              type="text"
              inputMode="numeric"
              placeholder={t('auth.emailCode')}
              value={passwordEmailCode}
              onChange={(e) => setPasswordEmailCode(e.target.value.trim())}
              className="min-w-0 flex-1 bg-transparent border-b border-mountain-grey py-2 px-1 text-ink focus:outline-none focus:border-ochre transition-colors placeholder-ink-light placeholder-opacity-50"
              required
            />
            <Button
              type="button"
              variant="ghost"
              size="sm"
              onClick={handleSendPasswordCode}
              disabled={passwordCodeSending || !passwordCaptchaId || !passwordCaptchaCode.trim()}
              loading={passwordCodeSending}
              className="h-10 shrink-0"
            >
              {passwordCodeSending ? t('auth.sendingEmailCode') : t('auth.sendEmailCode')}
            </Button>
          </div>
          <InlineNotice message={pwdError} />
          <InlineNotice message={pwdMsg} tone="success" />
          <div className="pt-4 text-center">
            <Button type="submit" variant="primary" size="md" loading={passwordSubmitting}>
              {passwordSubmitting ? t('profile.updatingPassword') : t('profile.updatePassword')}
            </Button>
          </div>
        </form>
      </section>
    </div>
  );
};

export default Profile;
