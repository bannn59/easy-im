import { useCallback, useEffect, useRef, useState } from 'react';
import { Link, Navigate, Outlet, useMatch } from 'react-router-dom';
import { useTranslation } from 'react-i18next';
import { listConversations, type Conversation } from '../api/conversations';
import { ApiError } from '../api/http';
import type { Message } from '../api/messages';
import { displayName } from '../features/chat/types';
import CreateGroupDialog from '../features/chat/CreateGroupDialog';
import { useRealtime } from '../realtime';
import { useSession } from './Session';

function sortConversations(items: Conversation[]): Conversation[] {
  return [...items].sort((a, b) => {
    const ta = a.last_message?.created_at || a.updated_at;
    const tb = b.last_message?.created_at || b.updated_at;
    return tb.localeCompare(ta);
  });
}

function formatListTime(iso: string | undefined): string {
  if (!iso) return '';
  const d = new Date(iso);
  if (Number.isNaN(d.getTime())) return '';
  const now = new Date();
  const sameDay =
    d.getFullYear() === now.getFullYear() &&
    d.getMonth() === now.getMonth() &&
    d.getDate() === now.getDate();
  if (sameDay) {
    return new Intl.DateTimeFormat(undefined, { hour: '2-digit', minute: '2-digit' }).format(d);
  }
  return new Intl.DateTimeFormat(undefined, { month: 'short', day: 'numeric' }).format(d);
}

function shortLocal(emailOrName: string): string {
  const s = emailOrName.trim();
  if (!s) return '';
  if (s.includes('@')) return s.split('@')[0] || s;
  return s;
}

function truncatePreviewBody(body: string, max = 120): string {
  const runes = [...body];
  return runes.length > max ? runes.slice(0, max).join('') : body;
}

function previewLine(
  c: Conversation,
  selfId: string | undefined,
  t: (k: string, o?: Record<string, string>) => string,
): string {
  const lm = c.last_message;
  if (!lm) return '';
  const body = lm.body;
  if (lm.sender_id === selfId) {
    return t('workspace.youPreview', { body });
  }
  // Group: prefix short sender name only when we have an email (avoid raw UUID).
  const isGroup = (c.member_count ?? c.members?.length ?? 0) > 2;
  if (isGroup) {
    const who = shortLocal(lm.sender_email || '');
    return who ? `${who}: ${body}` : body;
  }
  return body;
}

/** Mirror ConversationRoom title: explicit → group label → peer short name → untitled. */
function conversationListTitle(
  c: Conversation,
  selfId: string | undefined,
  t: (k: string) => string,
): string {
  const explicit = c.title?.trim();
  if (explicit) return explicit;
  const members = c.members ?? [];
  const count = c.member_count ?? members.length;
  if (count > 2) return t('chat.groupUntitled');
  const peer = members.find((m) => m.id !== selfId);
  if (peer) return displayName(peer.email, peer.display_name);
  return t('common.untitled');
}

export function AppShell() {
  const session = useSession();
  const roomMatch = useMatch('/app/c/:id');
  const activeId = roomMatch?.params.id;
  const activeIdRef = useRef(activeId);
  activeIdRef.current = activeId;
  const { t } = useTranslation();
  const [items, setItems] = useState<Conversation[]>([]);
  const [loadingList, setLoadingList] = useState(false);
  const [listError, setListError] = useState<string | null>(null);
  const [showCreateGroup, setShowCreateGroup] = useState(false);

  const refresh = useCallback(async () => {
    if (!session.user) return;
    setLoadingList(true);
    setListError(null);
    try {
      const res = await listConversations();
      setItems(sortConversations(res.conversations ?? []));
    } catch (err) {
      setListError(err instanceof ApiError ? err.message : t('common.failedToLoad'));
    } finally {
      setLoadingList(false);
    }
  }, [session.user, t]);

  useEffect(() => {
    if (session.user) {
      void refresh();
    }
  }, [session.user, refresh]);

  // Workspace-level realtime: patch list preview / unread (stable socket; active room via ref).
  const selfId = session.user?.id;
  const selfEmail = session.user?.email;
  useRealtime({
    onMessageCreated: (m: Message) => {
      setItems((prev) => {
        const idx = prev.findIndex((c) => c.id === m.conversation_id);
        if (idx < 0) return prev;
        const cur = prev[idx];
        const inRoom = activeIdRef.current === m.conversation_id;
        let unread = cur.unread_count ?? 0;
        if (inRoom) {
          unread = 0;
        } else if (m.sender_id !== selfId) {
          unread += 1;
        }
        // Preserve sender_email when possible (WS payload has no email).
        let senderEmail: string | null | undefined;
        if (m.sender_id === selfId) {
          senderEmail = selfEmail;
        } else if (m.sender_id === cur.last_message?.sender_id) {
          senderEmail = cur.last_message?.sender_email;
        }
        const next: Conversation = {
          ...cur,
          updated_at: m.created_at,
          last_message: {
            seq: m.seq,
            body: truncatePreviewBody(m.body),
            sender_id: m.sender_id,
            sender_email: senderEmail,
            created_at: m.created_at,
          },
          unread_count: unread,
        };
        const copy = prev.slice();
        copy[idx] = next;
        return sortConversations(copy);
      });
    },
    onMessageEdited: (m: Message) => {
      setItems((prev) =>
        prev.map((c) =>
          c.id === m.conversation_id && c.last_message?.seq === m.seq
            ? { ...c, last_message: { ...c.last_message, body: m.body } }
            : c,
        ),
      );
    },
    onMessageRecalled: (m: Message) => {
      setItems((prev) =>
        prev.map((c) =>
          c.id === m.conversation_id && c.last_message?.seq === m.seq
            ? { ...c, last_message: { ...c.last_message, body: t('chat.recalledPreview') } }
            : c,
        ),
      );
    },
  });

  // When entering a room, zero badge optimistically; room will mark-read.
  useEffect(() => {
    if (!activeId) return;
    setItems((prev) =>
      prev.map((c) => (c.id === activeId ? { ...c, unread_count: 0 } : c)),
    );
  }, [activeId]);

  if (session.loading) {
    return (
      <section className="page">
        <p className="loading">{t('workspace.loadingSession')}</p>
      </section>
    );
  }
  if (!session.user) {
    return <Navigate to="/login" replace />;
  }

  return (
    <div className="workspace">
      <aside className="workspace__side" aria-label={t('workspace.conversationsAria')}>
        <div className="workspace__side-head">
          <p className="page__eyebrow">{t('workspace.conversations')}</p>
          {session.user.display_name ? (
            <p className="muted">{session.user.display_name}</p>
          ) : null}
          <p className="muted">{session.user.email}</p>
          <p className="muted workspace__open-hint">{t('workspace.openFromFriends')}</p>
        </div>

        {listError && (
          <p className="err" role="alert">
            {listError}
          </p>
        )}
        {loadingList && <p className="loading">{t('common.loading')}</p>}

        <ul className="workspace__list">
          {items.map((c) => {
            const active = activeId === c.id;
            const unread = c.unread_count ?? 0;
            const time = formatListTime(c.last_message?.created_at || c.updated_at);
            const preview = previewLine(c, session.user?.id, t);
            return (
              <li key={c.id}>
                <Link
                  to={`/app/c/${c.id}`}
                  className={`workspace__item${active ? ' workspace__item--active' : ''}`}
                  aria-current={active ? 'page' : undefined}
                >
                  <div className="workspace__item-top">
                    <span className="workspace__item-title">
                      {conversationListTitle(c, session.user?.id, t)}
                    </span>
                    <span className="workspace__item-time muted">{time}</span>
                  </div>
                  <div className="workspace__item-bottom">
                    <span className="workspace__item-preview muted">{preview || c.id.slice(0, 8)}</span>
                    {unread > 0 && (
                      <span className="workspace__badge" aria-label={t('workspace.unreadAria', { count: unread })}>
                        {unread > 99 ? '99+' : unread}
                      </span>
                    )}
                  </div>
                </Link>
              </li>
            );
          })}
          {!loadingList && items.length === 0 && (
            <li className="muted">{t('workspace.noConversations')}</li>
          )}
        </ul>

        <div className="workspace__side-actions">
          <button type="button" className="btn btn--ghost" onClick={() => setShowCreateGroup(true)}>
            {t('chat.create')}
          </button>
          <Link to="/settings" className="btn btn--ghost">
            {t('nav.settings')}
          </Link>
          <button type="button" className="btn btn--ghost" onClick={() => session.logout()}>
            {t('workspace.signOut')}
          </button>
        </div>
      </aside>
      <div className="workspace__main">
        <Outlet />
      </div>
      {showCreateGroup && (
        <CreateGroupDialog onClose={() => setShowCreateGroup(false)} onCreated={() => void refresh()} />
      )}
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
