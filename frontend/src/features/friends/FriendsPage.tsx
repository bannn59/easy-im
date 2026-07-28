import { useCallback, useEffect, useState, type FormEvent } from 'react';
import { Navigate } from 'react-router-dom';
import { useTranslation } from 'react-i18next';
import {
  acceptFriendRequest,
  listFriends,
  listIncomingFriendRequests,
  rejectFriendRequest,
  sendFriendRequest,
  type FriendRequest,
} from '../../api/friends';
import type { PublicUser } from '../../api/auth';
import { ApiError } from '../../api/http';
import { useSession } from '../../app/Session';

export function FriendsPage() {
  const session = useSession();
  const { t } = useTranslation();
  const [email, setEmail] = useState('');
  const [sending, setSending] = useState(false);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [notice, setNotice] = useState<string | null>(null);
  const [incoming, setIncoming] = useState<FriendRequest[]>([]);
  const [friends, setFriends] = useState<PublicUser[]>([]);
  const [actingId, setActingId] = useState<string | null>(null);

  const refresh = useCallback(async () => {
    if (!session.token) return;
    setLoading(true);
    setError(null);
    try {
      const [inc, fr] = await Promise.all([
        listIncomingFriendRequests(session.token),
        listFriends(session.token),
      ]);
      setIncoming(inc.requests ?? []);
      setFriends(fr.friends ?? []);
    } catch (err) {
      setError(err instanceof ApiError ? err.message : t('common.failedToLoad'));
    } finally {
      setLoading(false);
    }
  }, [session.token, t]);

  useEffect(() => {
    if (session.user && session.token) {
      void refresh();
    }
  }, [session.user, session.token, refresh]);

  if (session.loading) {
    return (
      <section className="page">
        <p className="loading">{t('workspace.loadingSession')}</p>
      </section>
    );
  }
  if (!session.user || !session.token) {
    return <Navigate to="/login" replace />;
  }

  async function onSend(e: FormEvent) {
    e.preventDefault();
    if (!session.token) return;
    setSending(true);
    setError(null);
    setNotice(null);
    try {
      await sendFriendRequest(session.token, email.trim());
      setEmail('');
      setNotice(t('friends.requestSent'));
      await refresh();
    } catch (err) {
      setError(err instanceof ApiError ? err.message : t('common.requestFailed'));
    } finally {
      setSending(false);
    }
  }

  async function onAccept(id: string) {
    if (!session.token) return;
    setActingId(id);
    setError(null);
    setNotice(null);
    try {
      await acceptFriendRequest(session.token, id);
      setNotice(t('friends.accepted'));
      await refresh();
    } catch (err) {
      setError(err instanceof ApiError ? err.message : t('common.requestFailed'));
    } finally {
      setActingId(null);
    }
  }

  async function onReject(id: string) {
    if (!session.token) return;
    setActingId(id);
    setError(null);
    setNotice(null);
    try {
      await rejectFriendRequest(session.token, id);
      setNotice(t('friends.rejected'));
      await refresh();
    } catch (err) {
      setError(err instanceof ApiError ? err.message : t('common.requestFailed'));
    } finally {
      setActingId(null);
    }
  }

  return (
    <section className="page friends">
      <p className="page__eyebrow">{t('friends.eyebrow')}</p>
      <h1 className="page__title">{t('friends.title')}</h1>
      <p className="page__lead">{t('friends.lead')}</p>

      <form className="friends__send" onSubmit={onSend}>
        <div className="field">
          <label className="field__label" htmlFor="friend-email">
            {t('friends.emailLabel')}
          </label>
          <input
            id="friend-email"
            className="field__input"
            type="email"
            autoComplete="email"
            placeholder={t('friends.emailPlaceholder')}
            value={email}
            onChange={(ev) => setEmail(ev.target.value)}
            required
          />
        </div>
        <button className="btn" type="submit" disabled={sending || !email.trim()}>
          {sending ? t('friends.sending') : t('friends.sendRequest')}
        </button>
      </form>

      {error && (
        <p className="err" role="alert">
          {error}
        </p>
      )}
      {notice && (
        <p className="ok" role="status">
          {notice}
        </p>
      )}
      {loading && <p className="loading">{t('common.loading')}</p>}

      <section className="friends__section" aria-labelledby="friends-incoming-heading">
        <h2 id="friends-incoming-heading" className="friends__heading">
          {t('friends.incoming')}
        </h2>
        <ul className="friends__list">
          {incoming.map((req) => {
            const label = req.from_user?.email || req.from_user_id;
            const busy = actingId === req.id;
            return (
              <li key={req.id} className="friends__row">
                <span className="friends__email">{label}</span>
                <span className="friends__actions">
                  <button
                    type="button"
                    className="btn"
                    disabled={busy}
                    onClick={() => void onAccept(req.id)}
                  >
                    {t('friends.accept')}
                  </button>
                  <button
                    type="button"
                    className="btn btn--ghost"
                    disabled={busy}
                    onClick={() => void onReject(req.id)}
                  >
                    {t('friends.reject')}
                  </button>
                </span>
              </li>
            );
          })}
          {!loading && incoming.length === 0 && (
            <li className="muted">{t('friends.noIncoming')}</li>
          )}
        </ul>
      </section>

      <section className="friends__section" aria-labelledby="friends-list-heading">
        <h2 id="friends-list-heading" className="friends__heading">
          {t('friends.list')}
        </h2>
        <ul className="friends__list">
          {friends.map((f) => (
            <li key={f.id} className="friends__row">
              <span className="friends__email">{f.email}</span>
            </li>
          ))}
          {!loading && friends.length === 0 && (
            <li className="muted">{t('friends.noFriends')}</li>
          )}
        </ul>
      </section>
    </section>
  );
}
