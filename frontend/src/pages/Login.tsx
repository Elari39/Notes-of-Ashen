import React, { useState } from 'react';
import { useNavigate, Link, useLocation } from 'react-router-dom';
import { login } from '../api/auth';
import { useAuthStore } from '../store/auth';
import { usePreferenceStore } from '../store/preferences';
import { useSiteSettingsStore } from '../store/siteSettings';
import InlineNotice from '../components/InlineNotice';
import { getErrorMessage } from '../utils/error';
import { translate } from '../i18n';

const Login: React.FC = () => {
  const language = usePreferenceStore((state) => state.language);
  const registrationEnabled = useSiteSettingsStore((state) => state.registrationEnabled);
  const [account, setAccount] = useState('');
  const [password, setPassword] = useState('');
  const [error, setError] = useState('');
  const [submitting, setSubmitting] = useState(false);
  const navigate = useNavigate();
  const location = useLocation();
  const { setAuth, fetchUser } = useAuthStore();
  const t = (key: Parameters<typeof translate>[1]) => translate(language, key);
  const from = (location.state as { from?: Location } | null)?.from;

  const handleLogin = async (e: React.FormEvent) => {
    e.preventDefault();
    setError('');
    setSubmitting(true);
    try {
      const res = await login({ account, password });
      const token = res.data.accessToken;
      localStorage.setItem('refreshToken', res.data.refreshToken);
      setAuth(null, token);
      await fetchUser();
      const redirectTo = from ? `${from.pathname}${from.search}${from.hash}` : '/';
      navigate(redirectTo, { replace: true });
    } catch (err: unknown) {
      setError(getErrorMessage(err, t('auth.loginError')));
    } finally {
      setSubmitting(false);
    }
  };

  return (
    <div className="flex-grow flex items-center justify-center">
      <div className="w-full max-w-sm">
        <h1 className="text-3xl font-bold text-ink mb-12 text-center tracking-widest">{t('auth.loginTitle')}</h1>
        <form onSubmit={handleLogin} className="space-y-8">
          <div>
            <input
              type="text"
              placeholder={t('auth.accountOrEmail')}
              value={account}
              onChange={(e) => setAccount(e.target.value)}
              className="w-full bg-transparent border-b border-mountain-grey py-2 px-1 text-ink focus:outline-none focus:border-ochre transition-colors placeholder-ink-light placeholder-opacity-50"
              required
            />
          </div>
          <div>
            <input
              type="password"
              placeholder={t('auth.password')}
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
              disabled={submitting}
              className="w-full border border-ink text-ink py-3 hover:bg-ink hover:text-paper transition-colors duration-300 tracking-widest disabled:opacity-50 disabled:cursor-not-allowed"
            >
              {submitting ? t('auth.loginSubmitting') : t('auth.loginSubmit')}
            </button>
          </div>
          {registrationEnabled && (
            <div className="text-center text-sm text-ink-light opacity-70 mt-4">
              {t('auth.noAccount')} <Link to="/register" className="hover:text-ochre transition-colors">{t('auth.goRegister')}</Link>
            </div>
          )}
        </form>
      </div>
    </div>
  );
};

export default Login;
