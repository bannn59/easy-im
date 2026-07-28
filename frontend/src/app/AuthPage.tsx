import { useState, type FormEvent } from 'react';
import { Link, Navigate, useNavigate } from 'react-router-dom';
import { useTranslation } from 'react-i18next';
import { ApiError } from '../api/http';
import { useSession } from './Session';

type Mode = 'login' | 'register';

export function AuthPage({ mode }: { mode: Mode }) {
  const session = useSession();
  const navigate = useNavigate();
  const { t } = useTranslation();
  const [email, setEmail] = useState('');
  const [password, setPassword] = useState('');
  const [error, setError] = useState<string | null>(null);
  const [submitting, setSubmitting] = useState(false);

  if (!session.loading && session.user) {
    return <Navigate to="/app" replace />;
  }

  async function onSubmit(e: FormEvent) {
    e.preventDefault();
    setError(null);
    setSubmitting(true);
    try {
      if (mode === 'login') {
        await session.login(email, password);
      } else {
        await session.register(email, password);
      }
      navigate('/app', { replace: true });
    } catch (err) {
      if (err instanceof ApiError) {
        setError(err.message);
      } else {
        setError(t('common.requestFailed'));
      }
    } finally {
      setSubmitting(false);
    }
  }

  const isLogin = mode === 'login';

  return (
    <section className="page auth">
      <p className="page__eyebrow">{t('auth.eyebrow')}</p>
      <h1 className="page__title">{isLogin ? t('auth.signInTitle') : t('auth.registerTitle')}</h1>
      <p className="auth__switch">
        {isLogin ? (
          <>
            {t('auth.switchLoginPrefix')}{' '}
            <Link to="/register">{t('auth.switchLoginLink')}</Link>
          </>
        ) : (
          <>
            {t('auth.switchRegisterPrefix')}{' '}
            <Link to="/login">{t('auth.switchRegisterLink')}</Link>
          </>
        )}
      </p>

      <form className="auth__form" onSubmit={onSubmit} noValidate>
        <div className="field">
          <label className="field__label" htmlFor="auth-email">
            {t('auth.email')}
          </label>
          <input
            id="auth-email"
            className="field__input"
            type="email"
            autoComplete="email"
            value={email}
            onChange={(ev) => setEmail(ev.target.value)}
            required
          />
        </div>
        <div className="field">
          <label className="field__label" htmlFor="auth-password">
            {t('auth.password')}
          </label>
          <input
            id="auth-password"
            className="field__input"
            type="password"
            autoComplete={isLogin ? 'current-password' : 'new-password'}
            value={password}
            onChange={(ev) => setPassword(ev.target.value)}
            minLength={8}
            required
          />
        </div>
        {error && (
          <p className="err" role="alert">
            {error}
          </p>
        )}
        <div className="page__actions">
          <button className="btn" type="submit" disabled={submitting}>
            {submitting
              ? t('auth.working')
              : isLogin
                ? t('auth.signIn')
                : t('auth.createAccount')}
          </button>
        </div>
      </form>
    </section>
  );
}
