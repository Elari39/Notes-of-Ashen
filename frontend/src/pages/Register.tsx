import React, { useState } from 'react';
import { useNavigate, Link } from 'react-router-dom';
import { register } from '../api/auth';
import { useAuthStore } from '../store/auth';
import { normalizeAvatarUrl } from '../utils/avatar';
import InlineNotice from '../components/InlineNotice';
import { getErrorMessage } from '../utils/error';

const Register: React.FC = () => {
  const [account, setAccount] = useState('');
  const [password, setPassword] = useState('');
  const [email, setEmail] = useState('');
  const [nickname, setNickname] = useState('');
  const [avatarUrl, setAvatarUrl] = useState('');
  const [error, setError] = useState('');
  const [submitting, setSubmitting] = useState(false);
  const navigate = useNavigate();
  const { setAuth, fetchUser } = useAuthStore();

  const handleRegister = async (e: React.FormEvent) => {
    e.preventDefault();
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
      setError(getErrorMessage(err, '注册失败，请检查填写信息'));
    } finally {
      setSubmitting(false);
    }
  };

  return (
    <div className="flex-grow flex items-center justify-center">
      <div className="w-full max-w-sm">
        <h1 className="text-3xl font-bold text-ink mb-12 text-center tracking-widest">初逢</h1>
        <form onSubmit={handleRegister} className="space-y-8">
          <div>
            <input 
              type="text" 
              placeholder="账号 (3-64字)" 
              value={account}
              onChange={(e) => setAccount(e.target.value)}
              className="w-full bg-transparent border-b border-mountain-grey py-2 px-1 text-ink focus:outline-none focus:border-ochre transition-colors placeholder-ink-light placeholder-opacity-50"
              required
            />
          </div>
          <div>
            <input 
              type="email" 
              placeholder="邮箱" 
              value={email}
              onChange={(e) => setEmail(e.target.value)}
              className="w-full bg-transparent border-b border-mountain-grey py-2 px-1 text-ink focus:outline-none focus:border-ochre transition-colors placeholder-ink-light placeholder-opacity-50"
              required
            />
          </div>
          <div>
            <input 
              type="text" 
              placeholder="昵称 (可选)" 
              value={nickname}
              onChange={(e) => setNickname(e.target.value)}
              className="w-full bg-transparent border-b border-mountain-grey py-2 px-1 text-ink focus:outline-none focus:border-ochre transition-colors placeholder-ink-light placeholder-opacity-50"
            />
          </div>
          <div>
            <input 
              type="text" 
              placeholder="真容 (Avatar URL, 可选)" 
              value={avatarUrl}
              onChange={(e) => setAvatarUrl(e.target.value)}
              className="w-full bg-transparent border-b border-mountain-grey py-2 px-1 text-ink focus:outline-none focus:border-ochre transition-colors placeholder-ink-light placeholder-opacity-50"
            />
          </div>
          <div>
            <input 
              type="password" 
              placeholder="密码 (最少8字)" 
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
              {submitting ? '结缘中...' : '结缘'}
            </button>
          </div>
          <div className="text-center text-sm text-ink-light opacity-70 mt-4">
            已有前缘？ <Link to="/login" className="hover:text-ochre transition-colors">去入卷</Link>
          </div>
        </form>
      </div>
    </div>
  );
};

export default Register;
