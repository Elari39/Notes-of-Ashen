import React, { useEffect, useMemo, useState } from 'react';
import { useNavigate, Link, useLocation, type Location } from 'react-router-dom';
import { login } from '../api/auth';
import { useAuthStore } from '../store/auth';
import { usePreferenceStore } from '../store/preferences';
import { useSiteSettingsStore } from '../store/siteSettings';
import CaptchaField from '../components/CaptchaField';
import FormField from '../components/FormField';
import InlineNotice from '../components/InlineNotice';
import Button from '../components/ui/Button';
import { translate } from '../i18n';
import { useFormValidation, type FieldRules } from '../hooks/useFormValidation';
import { useSubmit } from '../hooks/useSubmit';
import { useShallow } from 'zustand/react/shallow';

const Login: React.FC = () => {
  const language = usePreferenceStore((state) => state.language);
  const registrationEnabled = useSiteSettingsStore((state) => state.registrationEnabled);
  const [account, setAccount] = useState('');
  const [password, setPassword] = useState('');
  const [captchaId, setCaptchaId] = useState('');
  const [captchaCode, setCaptchaCode] = useState('');
  const [captchaReloadKey, setCaptchaReloadKey] = useState(0);
  const [showPassword, setShowPassword] = useState(false);
  const navigate = useNavigate();
  const location = useLocation();
  const { user, accessToken, isFetching, isInitialized, setAuth, fetchUser } = useAuthStore(
    useShallow((state) => ({
      user: state.user,
      accessToken: state.accessToken,
      isFetching: state.isFetching,
      isInitialized: state.isInitialized,
      setAuth: state.setAuth,
      fetchUser: state.fetchUser,
    })),
  );
  const t = (key: Parameters<typeof translate>[1]) => translate(language, key);
  const from = (location.state as { from?: Location } | null)?.from;
  const redirectTo = from && from.pathname !== '/login'
    ? `${from.pathname}${from.search}${from.hash}`
    : '/';

  const rules = useMemo<FieldRules<{ account: string; password: string }>>(() => ({
    account: [{ type: 'required' }],
    password: [{ type: 'required' }],
  }), []);
  const { errors, validateField, validate } = useFormValidation({ account, password }, rules);

  const { submit: submitLogin, submitting, error } = useSubmit({
    handler: async () => {
      const res = await login({ account, password, captchaId, captchaCode });
      const token = res.data.accessToken;
      // refreshToken 由后端 HttpOnly Cookie 下发，前端无需也无需读取。
      setAuth(null, token);
      await fetchUser();
    },
    errorFallback: t('auth.loginError'),
    onSuccess: () => {
      navigate(redirectTo, { replace: true });
    },
    onError: () => {
      // 登录失败需要刷新图形验证码（旧逻辑保持不变）
      setCaptchaReloadKey((value) => value + 1);
    },
  });

  useEffect(() => {
    if (!isInitialized || isFetching || !accessToken || !user) {
      return;
    }
    navigate(redirectTo, { replace: true });
  }, [accessToken, isFetching, isInitialized, navigate, redirectTo, user]);

  const handleLogin = (e: React.FormEvent) => {
    e.preventDefault();
    if (!validate()) {
      return;
    }
    void submitLogin();
  };

  if (!isInitialized || isFetching) {
    return (
      <div className="flex-grow flex items-center justify-center tracking-widest text-ink-light">
        {t('common.loadingAuth')}
      </div>
    );
  }

  return (
    <div className="flex-grow flex items-center justify-center">
      <div className="w-full max-w-sm">
        <h1 className="text-3xl font-bold text-ink mb-12 text-center tracking-widest">{t('auth.loginTitle')}</h1>
        <form onSubmit={handleLogin} className="space-y-8">
          <FormField id="login-account" label={t('auth.accountOrEmail')} error={errors.account}>
            <input
              type="text"
              placeholder={t('auth.accountOrEmail')}
              value={account}
              onChange={(e) => setAccount(e.target.value)}
              onBlur={() => validateField('account')}
              className="w-full bg-transparent border-b border-mountain-grey py-2 px-1 text-ink focus:outline-none focus:border-ochre transition-colors placeholder-ink-light placeholder-opacity-50"
              required
            />
          </FormField>
          <div>
            <label htmlFor="login-password" className="mb-2 block text-xs tracking-widest text-ink-light">
              {t('auth.password')}
            </label>
            <div className="relative">
              <input
                id="login-password"
                type={showPassword ? 'text' : 'password'}
                placeholder={t('auth.password')}
                value={password}
                onChange={(e) => setPassword(e.target.value)}
                onBlur={() => validateField('password')}
                aria-invalid={errors.password ? true : undefined}
                aria-describedby={errors.password ? 'login-password-error' : undefined}
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
              <p id="login-password-error" role="alert" className="mt-2 border-l-2 border-ember bg-[var(--notice-bg)] px-3 py-2 text-xs text-ember">
                {errors.password}
              </p>
            )}
          </div>
          <CaptchaField
            purpose="login"
            captchaId={captchaId}
            captchaCode={captchaCode}
            onCaptchaIdChange={setCaptchaId}
            onCaptchaCodeChange={setCaptchaCode}
            reloadKey={captchaReloadKey}
          />
          <InlineNotice message={error} />
          <div className="pt-4">
            <Button
              type="submit"
              variant="primary"
              size="lg"
              loading={submitting}
              fullWidth
            >
              {submitting ? t('auth.loginSubmitting') : t('auth.loginSubmit')}
            </Button>
          </div>
          {registrationEnabled && (
            <div className="text-center text-sm text-ink-light opacity-70 mt-4">
              {t('auth.noAccount')} <Link to="/register" className="hover:text-ochre transition-colors">{t('auth.goRegister')}</Link>
            </div>
          )}
          <div className="text-center text-sm text-ink-light opacity-70 mt-4">
            <Link to="/forgot-password" className="hover:text-ochre transition-colors">{t('auth.forgotPassword')}</Link>
          </div>
        </form>
      </div>
    </div>
  );
};

export default Login;
