import { useCallback, useEffect, useState, type FormEvent } from 'react';
import { Link, Navigate, Outlet, useNavigate, useMatch } from 'react-router-dom';
import { useTranslation } from 'react-i18next';
import {
  createConversation,
  listConversations,
  type Conversation,
} from '../api/conversations';
import { ApiError } from '../api/http';
import { useSession } from './Session';

export function AppShell() {
  const session = useSession();
  const navigate = useNavigate();
  const roomMatch = useMatch('/app/c/:id');
  const activeId = roomMatch?.params.id;
  const { t } = useTranslation();
  const [items, setItems] = useState<Conversation[]>([]);
  const [loadingList, setLoadingList] = useState(false);
  const [listError, setListError] = useState<string | null>(null);
  const [memberEmail, setMemberEmail] = useState('');
  const [title, setTitle] = useState('');
  const [creating, setCreating] = useState(false);

  const refresh = useCallback(async () => {
    if (!session.token) return;
    setLoadingList(true);
    setListError(null);
    try {
      const res = await listConversations(session.token);
      setItems(res.conversations ?? []);
    } catch (err) {
      setListError(err instanceof ApiError ? err.message : t('common.failedToLoad'));
    } finally {
      setLoadingList(false);
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

  async function onCreate(e: FormEvent) {
    e.preventDefault();
    if (!session.token) return;
    setCreating(true);
    setListError(null);
    try {
      const emails = memberEmail
        .split(',')
        .map((s) => s.trim())
        .filter(Boolean);
      const c = await createConversation(session.token, {
        title: title.trim() || undefined,
        member_emails: emails,
      });
      setMemberEmail('');
      setTitle('');
      await refresh();
      navigate(`/app/c/${c.id}`);
    } catch (err) {
      setListError(err instanceof ApiError ? err.message : t('common.createFailed'));
    } finally {
      setCreating(false);
    }
  }

  return (
    <div className="workspace">
      <aside className="workspace__side" aria-label={t('workspace.conversationsAria')}>
        <div className="workspace__side-head">
          <p className="page__eyebrow">{t('workspace.conversations')}</p>
          <p className="muted">{session.user.email}</p>
        </div>

        <form className="workspace__create" onSubmit={onCreate}>
          <div className="field">
            <label className="field__label" htmlFor="member-emails">
              {t('workspace.memberEmails')}
            </label>
            <input
              id="member-emails"
              className="field__input"
              placeholder={t('workspace.memberEmailsPlaceholder')}
              value={memberEmail}
              onChange={(ev) => setMemberEmail(ev.target.value)}
            />
          </div>
          <div className="field">
            <label className="field__label" htmlFor="conv-title">
              {t('workspace.titleOptional')}
            </label>
            <input
              id="conv-title"
              className="field__input"
              value={title}
              onChange={(ev) => setTitle(ev.target.value)}
            />
          </div>
          <button className="btn" type="submit" disabled={creating}>
            {creating ? t('workspace.creating') : t('workspace.newConversation')}
          </button>
        </form>

        {listError && (
          <p className="err" role="alert">
            {listError}
          </p>
        )}
        {loadingList && <p className="loading">{t('common.loading')}</p>}

        <ul className="workspace__list">
          {items.map((c) => {
            const active = activeId === c.id;
            return (
              <li key={c.id}>
                <Link
                  to={`/app/c/${c.id}`}
                  className={`workspace__item${active ? ' workspace__item--active' : ''}`}
                  aria-current={active ? 'page' : undefined}
                >
                  <span>{c.title?.trim() ? c.title : t('common.untitled')}</span>
                  <span className="muted">{c.id.slice(0, 8)}</span>
                </Link>
              </li>
            );
          })}
          {!loadingList && items.length === 0 && (
            <li className="muted">{t('workspace.noConversations')}</li>
          )}
        </ul>

        <button type="button" className="btn btn--ghost" onClick={() => session.logout()}>
          {t('workspace.signOut')}
        </button>
      </aside>
      <div className="workspace__main">
        <Outlet />
      </div>
    </div>
  );
}

export function ConversationHome() {
  const { t } = useTranslation();
  return (
    <section className="page room room--empty">
      <p className="page__eyebrow">{t('workspace.homeEyebrow')}</p>
      <h1 className="page__title">{t('workspace.homeTitle')}</h1>
      <p className="page__lead">{t('workspace.homeLead')}</p>
    </section>
  );
}
