import React, { useState } from 'react';
import { useNavigate, Link } from 'react-router-dom';
import { register } from '../api/auth';
import { useAuthStore } from '../store/auth';
import { usePreferenceStore } from '../store/preferences';
import { useSiteSettingsStore } from '../store/siteSettings';
import { normalizeAvatarUrl } from '../utils/avatar';
import InlineNotice from '../components/InlineNotice';
import { getErrorMessage } from '../utils/error';
import { translate } from '../i18n';

const Register: React.FC = () => {
  const language = usePreferenceStore((state) => state.language);
  const { registrationEnabled, hasLoaded, isLoading } = useSiteSettingsStore();
  const [account, setAccount] = useState('');
  const [password, setPassword] = useState('');
  const [email, setEmail] = useState('');
  const [nickname, setNickname] = useState('');
  const [avatarUrl, setAvatarUrl] = useState('');
  const [error, setError] = useState('');
  const [submitting, setSubmitting] = useState(false);
  const navigate = useNavigate();
  const { setAuth, fetchUser } = useAuthStore();
  const t = (key: Parameters<typeof translate>[1]) => translate(language, key);

  const handleRegister = async (e: React.FormEvent) => {
    e.preventDefault();
    if (hasLoaded && !registrationEnabled) {
      setError(t('auth.registrationDisabled'));
      return;
    }
    setError('');
    setSubmitting(true);
    try {
      const res = await register({
        account: account.trim(),
        password,
        email: email.trim(),
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
    } finally {
      setSubmitting(false);
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
          <div>
            <input
              type="text"
              placeholder={t('auth.accountWithHint')}
              value={account}
              onChange={(e) => setAccount(e.target.value)}
              className="w-full bg-transparent border-b border-mountain-grey py-2 px-1 text-ink focus:outline-none focus:border-ochre transition-colors placeholder-ink-light placeholder-opacity-50"
              required
            />
          </div>
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
            <input
              type="password"
              placeholder={t('auth.passwordWithHint')}
              value={password}
              onChange={(e) => setPassword(e.target.value)}
              className="w-full bg-transparent border-b border-mountain-grey py-2 px-1 text-ink focus:outline-none focus:border-ochre transition-colors placeholder-ink-light placeholder-opacity-50"
              required
            />
          </div>
          <InlineNotice message={error} />
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
