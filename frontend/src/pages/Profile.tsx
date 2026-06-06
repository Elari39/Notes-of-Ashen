import React, { useEffect, useState } from 'react';
import { useAuthStore } from '../store/auth';
import { updateCurrentUser, updatePassword } from '../api/user';
import { isHttpAvatarUrl, normalizeAvatarUrl } from '../utils/avatar';
import InlineNotice from '../components/InlineNotice';
import { getErrorMessage } from '../utils/error';

const Profile: React.FC = () => {
  const { user, fetchUser } = useAuthStore();
  const [nickname, setNickname] = useState(user?.nickname || '');
  const [email, setEmail] = useState(user?.email || '');
  const [avatarUrl, setAvatarUrl] = useState(user?.avatarUrl || '');
  
  const [oldPassword, setOldPassword] = useState('');
  const [newPassword, setNewPassword] = useState('');
  const [pwdError, setPwdError] = useState('');
  const [pwdMsg, setPwdMsg] = useState('');
  const [msg, setMsg] = useState('');
  const [profileSubmitting, setProfileSubmitting] = useState(false);
  const [passwordSubmitting, setPasswordSubmitting] = useState(false);
  const previewAvatarUrl = normalizeAvatarUrl(avatarUrl);
  const shouldShowAvatarPreview = isHttpAvatarUrl(previewAvatarUrl);

  useEffect(() => {
    setNickname(user?.nickname || '');
    setEmail(user?.email || '');
    setAvatarUrl(user?.avatarUrl || '');
  }, [user]);

  const handleUpdateProfile = async (e: React.FormEvent) => {
    e.preventDefault();
    setMsg('');
    setProfileSubmitting(true);
    try {
      await updateCurrentUser({
        nickname: nickname.trim(),
        email: email.trim(),
        avatarUrl: normalizeAvatarUrl(avatarUrl),
      });
      await fetchUser();
      setMsg('资料已更新');
    } catch (err: unknown) {
      setMsg(getErrorMessage(err, '更新失败'));
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
      setPwdMsg('密码已更新');
    } catch (err: unknown) {
      setPwdError(getErrorMessage(err, '更新失败'));
    } finally {
      setPasswordSubmitting(false);
    }
  };

  return (
    <div className="max-w-md mx-auto w-full space-y-16">
      <section>
        <h2 className="text-2xl font-bold text-ink mb-8 tracking-widest text-center">命籍 (资料)</h2>
        <form onSubmit={handleUpdateProfile} className="space-y-6">
          <div>
            <label className="block text-sm text-ink-light opacity-70 mb-2">账文</label>
            <input 
              type="text" 
              value={user?.account} 
              disabled 
              className="w-full bg-transparent border-b border-mountain-grey py-2 px-1 text-ink-light opacity-50 cursor-not-allowed"
            />
          </div>
          <div>
            <label className="block text-sm text-ink-light opacity-70 mb-2">别号</label>
            <input 
              type="text" 
              value={nickname}
              onChange={e => setNickname(e.target.value)}
              className="w-full bg-transparent border-b border-mountain-grey py-2 px-1 text-ink focus:outline-none focus:border-ochre transition-colors"
            />
          </div>
          <div>
            <label className="block text-sm text-ink-light opacity-70 mb-2">飞书 (Email)</label>
            <input 
              type="email" 
              value={email}
              onChange={e => setEmail(e.target.value)}
              className="w-full bg-transparent border-b border-mountain-grey py-2 px-1 text-ink focus:outline-none focus:border-ochre transition-colors"
            />
          </div>
          <div>
            <label className="block text-sm text-ink-light opacity-70 mb-2">真容 (Avatar URL)</label>
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
          <InlineNotice message={msg} tone={msg === '资料已更新' ? 'success' : 'error'} />
          <div className="pt-4 text-center">
            <button type="submit" disabled={profileSubmitting} className="px-8 py-2 border border-ink text-ink hover:bg-ink hover:text-paper transition-colors tracking-widest disabled:opacity-50 disabled:cursor-not-allowed">
              {profileSubmitting ? '修纂中...' : '修纂'}
            </button>
          </div>
        </form>
      </section>

      <div className="w-24 h-px bg-mountain-grey mx-auto opacity-50"></div>

      <section>
        <h2 className="text-2xl font-bold text-ink mb-8 tracking-widest text-center">秘钥 (密码)</h2>
        <form onSubmit={handleUpdatePassword} className="space-y-6">
          <div>
            <input 
              type="password" 
              placeholder="旧梦 (原密码)"
              value={oldPassword}
              onChange={e => setOldPassword(e.target.value)}
              required
              className="w-full bg-transparent border-b border-mountain-grey py-2 px-1 text-ink focus:outline-none focus:border-ochre transition-colors placeholder-ink-light placeholder-opacity-50"
            />
          </div>
          <div>
            <input 
              type="password" 
              placeholder="新声 (新密码)"
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
              {passwordSubmitting ? '更易中...' : '更易'}
            </button>
          </div>
        </form>
      </section>
    </div>
  );
};

export default Profile;
