import { useCallback, useEffect, useState, type FormEvent } from 'react';
import { Link, Navigate, Outlet, useNavigate, useParams } from 'react-router-dom';
import { useTranslation } from 'react-i18next';
import {
  createConversation,
  getConversation,
  listConversations,
  type Conversation,
} from '../api/conversations';
import { listMessages, sendMessage, type Message } from '../api/messages';
import { ApiError } from '../api/http';
import { connectRealtime } from '../realtime';
import { useSession } from './Session';

export function AppShell() {
  const session = useSession();
  const navigate = useNavigate();
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
          {items.map((c) => (
            <li key={c.id}>
              <Link to={`/app/c/${c.id}`} className="workspace__item">
                <span>{c.title?.trim() ? c.title : t('common.untitled')}</span>
                <span className="muted">{c.id.slice(0, 8)}</span>
              </Link>
            </li>
          ))}
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
    <section className="page">
      <p className="page__eyebrow">{t('workspace.homeEyebrow')}</p>
      <h1 className="page__title">{t('workspace.homeTitle')}</h1>
      <p className="page__lead">{t('workspace.homeLead')}</p>
    </section>
  );
}

function newClientMsgId(): string {
  if (typeof crypto !== 'undefined' && 'randomUUID' in crypto) {
    return crypto.randomUUID();
  }
  return `c-${Date.now()}-${Math.random().toString(16).slice(2)}`;
}

export function ConversationRoom() {
  const { id } = useParams();
  const session = useSession();
  const { t } = useTranslation();
  const [conv, setConv] = useState<Conversation | null>(null);
  const [messages, setMessages] = useState<Message[]>([]);
  const [error, setError] = useState<string | null>(null);
  const [loading, setLoading] = useState(true);
  const [text, setText] = useState('');
  const [sending, setSending] = useState(false);

  const loadMessages = useCallback(async () => {
    if (!session.token || !id) return;
    const res = await listMessages(session.token, id, { limit: 100 });
    setMessages(res.messages ?? []);
  }, [session.token, id]);

  useEffect(() => {
    if (!session.token || !id) return;
    let cancelled = false;
    setLoading(true);
    setError(null);
    Promise.all([getConversation(session.token, id), listMessages(session.token, id, { limit: 100 })])
      .then(([c, msgRes]) => {
        if (!cancelled) {
          setConv(c);
          setMessages(msgRes.messages ?? []);
        }
      })
      .catch((err: unknown) => {
        if (!cancelled) {
          setConv(null);
          setMessages([]);
          setError(err instanceof ApiError ? err.message : t('common.failedToLoad'));
        }
      })
      .finally(() => {
        if (!cancelled) setLoading(false);
      });
    return () => {
      cancelled = true;
    };
  }, [id, session.token, t]);

  // Realtime: merge pushed messages; keep a slow poll as fallback.
  useEffect(() => {
    if (!session.token || !id || !conv) return;
    const stop = connectRealtime(session.token, {
      onMessageCreated: (m) => {
        if (m.conversation_id !== id) return;
        setMessages((prev) => {
          if (prev.some((x) => x.id === m.id || x.client_msg_id === m.client_msg_id)) {
            return prev;
          }
          return [...prev, m].sort((a, b) => a.seq - b.seq);
        });
      },
    });
    const timer = window.setInterval(() => {
      void loadMessages().catch(() => undefined);
    }, 15000);
    return () => {
      stop();
      window.clearInterval(timer);
    };
  }, [session.token, id, conv, loadMessages]);

  async function onSend(e: FormEvent) {
    e.preventDefault();
    if (!session.token || !id || !text.trim()) return;
    setSending(true);
    setError(null);
    const clientMsgId = newClientMsgId();
    try {
      const m = await sendMessage(session.token, id, {
        body: text.trim(),
        client_msg_id: clientMsgId,
      });
      setText('');
      setMessages((prev) => {
        if (prev.some((x) => x.id === m.id)) return prev;
        return [...prev, m];
      });
    } catch (err) {
      setError(err instanceof ApiError ? err.message : t('common.sendFailed'));
    } finally {
      setSending(false);
    }
  }

  if (loading) {
    return <p className="loading">{t('workspace.loadingConversation')}</p>;
  }
  if (error && !conv) {
    return (
      <section className="page">
        <p className="err">{error}</p>
      </section>
    );
  }
  if (!conv) {
    return null;
  }

  const memberLabel = (senderId: string) => {
    const m = conv.members?.find((x) => x.id === senderId);
    return m?.email ?? senderId.slice(0, 8);
  };

  return (
    <section className="page room">
      <p className="page__eyebrow">{t('workspace.roomEyebrow')}</p>
      <h1 className="page__title">{conv.title?.trim() ? conv.title : t('common.untitled')}</h1>
      <p className="page__meta">
        {(conv.members ?? []).map((m) => m.email).join(' · ') || <code>{conv.id}</code>}
      </p>

      <ul className="msg-list" aria-live="polite">
        {messages.map((m) => {
          const mine = m.sender_id === session.user?.id;
          return (
            <li key={m.id} className={mine ? 'msg msg--mine' : 'msg'}>
              <div className="msg__meta">
                <span>{mine ? t('common.you') : memberLabel(m.sender_id)}</span>
                <span className="muted">#{m.seq}</span>
              </div>
              <p className="msg__body">{m.body}</p>
            </li>
          );
        })}
        {messages.length === 0 && <li className="muted">{t('workspace.noMessages')}</li>}
      </ul>

      {error && (
        <p className="err" role="alert">
          {error}
        </p>
      )}

      <form className="composer" onSubmit={onSend}>
        <label className="field__label" htmlFor="msg-body">
          {t('workspace.messageLabel')}
        </label>
        <input
          id="msg-body"
          className="field__input"
          value={text}
          onChange={(ev) => setText(ev.target.value)}
          placeholder={t('workspace.messagePlaceholder')}
          autoComplete="off"
        />
        <button className="btn" type="submit" disabled={sending || !text.trim()}>
          {sending ? t('workspace.sending') : t('workspace.send')}
        </button>
      </form>
    </section>
  );
}
