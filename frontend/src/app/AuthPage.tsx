import { useState, type FormEvent } from 'react';
import { Link, Navigate, useNavigate } from 'react-router-dom';
import { ApiError } from '../api/http';
import { useSession } from './Session';

type Mode = 'login' | 'register';

export function AuthPage({ mode }: { mode: Mode }) {
  const session = useSession();
  const navigate = useNavigate();
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
        setError('Request failed');
      }
    } finally {
      setSubmitting(false);
    }
  }

  return (
    <section className="auth">
      <h1>{mode === 'login' ? 'Sign in' : 'Create account'}</h1>
      <p className="muted">
        {mode === 'login' ? (
          <>
            No account? <Link to="/register">Register</Link>
          </>
        ) : (
          <>
            Already registered? <Link to="/login">Sign in</Link>
          </>
        )}
      </p>
      <form className="auth__form" onSubmit={onSubmit}>
        <label>
          Email
          <input
            type="email"
            autoComplete="email"
            value={email}
            onChange={(ev) => setEmail(ev.target.value)}
            required
          />
        </label>
        <label>
          Password
          <input
            type="password"
            autoComplete={mode === 'login' ? 'current-password' : 'new-password'}
            value={password}
            onChange={(ev) => setPassword(ev.target.value)}
            minLength={8}
            required
          />
        </label>
        {error && <p className="err">{error}</p>}
        <button type="submit" disabled={submitting}>
          {submitting ? 'Please wait…' : mode === 'login' ? 'Sign in' : 'Register'}
        </button>
      </form>
    </section>
  );
}
