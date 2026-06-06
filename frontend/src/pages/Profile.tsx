import React, { useEffect, useState } from 'react';
import { useAuthStore } from '../store/auth';
import { usePreferenceStore } from '../store/preferences';
import { updateCurrentUser, updatePassword } from '../api/user';
import { isHttpAvatarUrl, normalizeAvatarUrl } from '../utils/avatar';
import InlineNotice from '../components/InlineNotice';
import { getErrorMessage } from '../utils/error';
import { translate } from '../i18n';

const Profile: React.FC = () => {
  const { user, fetchUser } = useAuthStore();
  const language = usePreferenceStore((state) => state.language);
  const [nickname, setNickname] = useState(user?.nickname || '');
  const [email, setEmail] = useState(user?.email || '');
  const [avatarUrl, setAvatarUrl] = useState(user?.avatarUrl || '');

  const [oldPassword, setOldPassword] = useState('');
  const [newPassword, setNewPassword] = useState('');
  const [pwdError, setPwdError] = useState('');
  const [pwdMsg, setPwdMsg] = useState('');
  const [msg, setMsg] = useState('');
  const [profileUpdated, setProfileUpdated] = useState(false);
  const [profileSubmitting, setProfileSubmitting] = useState(false);
  const [passwordSubmitting, setPasswordSubmitting] = useState(false);
  const previewAvatarUrl = normalizeAvatarUrl(avatarUrl);
  const shouldShowAvatarPreview = isHttpAvatarUrl(previewAvatarUrl);
  const t = (key: Parameters<typeof translate>[1]) => translate(language, key);

  useEffect(() => {
    setNickname(user?.nickname || '');
    setEmail(user?.email || '');
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
        avatarUrl: normalizeAvatarUrl(avatarUrl),
      });
      await fetchUser();
      setMsg(t('profile.updated'));
      setProfileUpdated(true);
    } catch (err: unknown) {
      setMsg(getErrorMessage(err, t('profile.updateError')));
    } finally {
      setProfileSubmitting(false);
    }
  };

  const handleUpdatePassword = async (e: React.FormEvent) => {
    e.preventDefault();
    setPwdError('');
    setPwdMsg('');
    setPasswordSubmitting(true);
    try {
      await updatePassword({ oldPassword, newPassword });
      setOldPassword('');
      setNewPassword('');
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
              value={user?.account}
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
            <button type="submit" disabled={profileSubmitting} className="px-8 py-2 border border-ink text-ink hover:bg-ink hover:text-paper transition-colors tracking-widest disabled:opacity-50 disabled:cursor-not-allowed">
              {profileSubmitting ? t('profile.updatingProfile') : t('profile.updateProfile')}
            </button>
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
          </div>
          <InlineNotice message={pwdError} />
          <InlineNotice message={pwdMsg} tone="success" />
          <div className="pt-4 text-center">
            <button type="submit" disabled={passwordSubmitting} className="px-8 py-2 border border-ink text-ink hover:bg-ink hover:text-paper transition-colors tracking-widest disabled:opacity-50 disabled:cursor-not-allowed">
              {passwordSubmitting ? t('profile.updatingPassword') : t('profile.updatePassword')}
            </button>
          </div>
        </form>
      </section>
    </div>
  );
};

export default Profile;
