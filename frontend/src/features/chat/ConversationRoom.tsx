import { useCallback, useEffect, useRef, useState, type FormEvent } from 'react';
import { useNavigate, useParams } from 'react-router-dom';
import { useTranslation } from 'react-i18next';
import { getConversation, markConversationRead, leaveGroup, kickGroupMember, transferGroupOwner, renameGroup, type Conversation } from '../../api/conversations';
import { editMessage, listMessages, recallMessage, sendMessage, type Message } from '../../api/messages';
import { ApiError } from '../../api/http';
import { useRealtime, sendFrame } from '../../realtime';
import { useSession } from '../../app/Session';
import { Composer, type ComposerReply } from './Composer';
import { MessageList } from './MessageList';
import AddMembersDialog from './AddMembersDialog';
import { displayName, mergeMessage, newClientMsgId, type ChatItem } from './types';

const NEAR_BOTTOM_PX = 80;

function toChatItem(m: Message, status: ChatItem['status'] = 'sent'): ChatItem {
  return {
    id: m.id,
    conversation_id: m.conversation_id,
    sender_id: m.sender_id,
    body: m.body,
    client_msg_id: m.client_msg_id,
    seq: m.seq,
    created_at: m.created_at,
    reply_to: m.reply_to ?? null,
    edited_at: m.edited_at ?? null,
    recalled_at: m.recalled_at ?? null,
    status,
  };
}

export function ConversationRoom() {
  const { id } = useParams();
  const navigate = useNavigate();
  const session = useSession();
  const { t } = useTranslation();
  const [conv, setConv] = useState<Conversation | null>(null);
  const [messages, setMessages] = useState<ChatItem[]>([]);
  const [error, setError] = useState<string | null>(null);
  const [loading, setLoading] = useState(true);
  const [text, setText] = useState('');
  const [sending, setSending] = useState(false);
  const [reply, setReply] = useState<ComposerReply | null>(null);
  const [peerReadSeq, setPeerReadSeq] = useState<Map<string, number>>(new Map());
  const [typingUsers, setTypingUsers] = useState<Set<string>>(new Set());
  const [presenceOverrides, setPresenceOverrides] = useState<Record<string, boolean>>({});
  const [showMembers, setShowMembers] = useState(false);
  const [memberNotice, setMemberNotice] = useState<string | null>(null);
  const [showAddMembers, setShowAddMembers] = useState(false);
  const [renaming, setRenaming] = useState(false);
  const [renameValue, setRenameValue] = useState('');
  const typingTimers = useRef<Map<string, number>>(new Map());
  const lastTypingSent = useRef(0);
  const listRef = useRef<HTMLUListElement>(null!);
  const stickToBottom = useRef(true);

  const memberLabel = useCallback(
    (senderId: string) => {
      if (senderId && senderId === session.user?.id) {
        return session.user.email;
      }
      const m = conv?.members?.find((x) => x.id === senderId);
      return m ? displayName(m.email, m.display_name) : senderId.slice(0, 8);
    },
    [conv, session.user],
  );

  const scrollToBottom = useCallback((force = false) => {
    const el = listRef.current;
    if (!el) return;
    if (!force && !stickToBottom.current) return;
    el.scrollTop = el.scrollHeight;
  }, []);

  const onListScroll = useCallback(() => {
    const el = listRef.current;
    if (!el) return;
    const dist = el.scrollHeight - el.scrollTop - el.clientHeight;
    stickToBottom.current = dist < NEAR_BOTTOM_PX;
  }, []);

  const loadMessages = useCallback(async () => {
    if (!session.user || !id) return;
    const res = await listMessages(id, { limit: 100 });
    setMessages((res.messages ?? []).map((m) => toChatItem(m)));
  }, [session.user, id]);

  useEffect(() => {
    if (!session.user || !id) return;
    let cancelled = false;
    setLoading(true);
    setError(null);
    setReply(null);
    setText('');
    stickToBottom.current = true;
    Promise.all([getConversation(id), listMessages(id, { limit: 100 })])
      .then(async ([c, msgRes]) => {
        if (cancelled) return;
        setConv(c);
        setMessages((msgRes.messages ?? []).map((m) => toChatItem(m)));
        // Self-only unread: opening the room marks read to head.
        try {
          await markConversationRead(id);
        } catch {
          // non-fatal: badge may clear on next list refresh
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
  }, [id, session.user, t]);

  useEffect(() => {
    if (!loading) {
      requestAnimationFrame(() => scrollToBottom(true));
    }
  }, [loading, messages.length, scrollToBottom]);

  useEffect(() => {
    const el = listRef.current;
    if (!el) return;
    el.addEventListener('scroll', onListScroll, { passive: true });
    return () => el.removeEventListener('scroll', onListScroll);
  }, [onListScroll, loading, conv]);

  // Room-level subscriptions ride the app-wide connection (useRealtime).
  const convId = id;
  const selfId = session.user?.id;
  useRealtime({
    onMessageCreated: (m) => {
      if (m.conversation_id !== convId) return;
      setMessages((prev) => mergeMessage(prev, toChatItem(m)));
      requestAnimationFrame(() => scrollToBottom(false));
      void markConversationRead(convId, m.seq).catch(() => undefined);
    },
    onMessageEdited: (m) => {
      if (m.conversation_id !== convId) return;
      setMessages((prev) => mergeMessage(prev, toChatItem(m)));
    },
    onMessageRecalled: (m) => {
      if (m.conversation_id !== convId) return;
      setMessages((prev) => mergeMessage(prev, toChatItem(m)));
    },
    onMessageRead: (data) => {
      if (data.conversation_id !== convId) return;
      setPeerReadSeq((prev) => {
        const next = new Map(prev);
        next.set(data.reader_id, Math.max(data.last_read_seq, next.get(data.reader_id) ?? 0));
        return next;
      });
    },
    onTypingStarted: (data) => {
      if (data.conversation_id !== convId || data.user_id === selfId) return;
      setTypingUsers((prev) => new Set(prev).add(data.user_id));
      // Client-side 4-second timeout
      const existing = typingTimers.current.get(data.user_id);
      if (existing) window.clearTimeout(existing);
      typingTimers.current.set(
        data.user_id,
        window.setTimeout(() => {
          setTypingUsers((prev) => {
            const next = new Set(prev);
            next.delete(data.user_id);
            return next;
          });
          typingTimers.current.delete(data.user_id);
        }, 4000),
      );
    },
    onTypingStopped: (data) => {
      if (data.conversation_id !== convId) return;
      setTypingUsers((prev) => {
        const next = new Set(prev);
        next.delete(data.user_id);
        return next;
      });
      const existing = typingTimers.current.get(data.user_id);
      if (existing) {
        window.clearTimeout(existing);
        typingTimers.current.delete(data.user_id);
      }
    },
    onPresenceChanged: ({ user_id, online }) => {
      setPresenceOverrides((prev) => ({ ...prev, [user_id]: online }));
    },
    onMembersChanged: (data) => {
      if (data.conversation_id !== id) return;
      // Re-fetch the conversation to pick up the fresh member list.
      if (session.user) {
        void getConversation(id).then(setConv).catch(() => undefined);
      }
    },
    onConversationRenamed: (data) => {
      if (data.conversation_id !== id) return;
      setConv((prev) => (prev ? { ...prev, title: data.title, updated_at: data.updated_at } : prev));
    },
  });

  // 15s polling fallback while the room is open.
  useEffect(() => {
    if (!session.user || !id) return;
    const timer = window.setInterval(() => {
      void loadMessages().catch(() => undefined);
    }, 15000);
    return () => window.clearInterval(timer);
  }, [session.user, id, loadMessages]);

  async function doSend(body: string, replyTo: ComposerReply | null, clientMsgId: string) {
    if (!session.user || !id) return;
    const trimmed = body.trim();
    if (!trimmed) return;

    const selfId = session.user.id;
    const convId = id;

    stickToBottom.current = true;
    setText('');
    setReply(null);
    setSending(true);
    setError(null);

    setMessages((prev) => {
      const lastSeq = prev.reduce((max, m) => (m.seq > max ? m.seq : max), 0);
      const pending: ChatItem = {
        id: `local:${clientMsgId}`,
        conversation_id: convId,
        sender_id: selfId,
        body: trimmed,
        client_msg_id: clientMsgId,
        seq: lastSeq + 0.001,
        created_at: new Date().toISOString(),
        reply_to: replyTo
          ? { id: replyTo.id, sender_id: replyTo.sender_id, body: replyTo.body }
          : null,
        status: 'pending',
        localKey: clientMsgId,
      };
      return mergeMessage(prev, pending);
    });
    requestAnimationFrame(() => scrollToBottom(true));

    try {
      const m = await sendMessage(convId, {
        body: trimmed,
        client_msg_id: clientMsgId,
        reply_to_message_id: replyTo?.id,
      });
      setMessages((prev) => mergeMessage(prev, toChatItem(m, 'sent')));
      requestAnimationFrame(() => scrollToBottom(true));
    } catch (err) {
      setMessages((prev) =>
        prev.map((x) =>
          x.client_msg_id === clientMsgId || x.localKey === clientMsgId
            ? { ...x, status: 'failed' as const }
            : x,
        ),
      );
      setError(err instanceof ApiError ? err.message : t('common.sendFailed'));
    } finally {
      setSending(false);
    }
  }

  function onSend() {
    void doSend(text, reply, newClientMsgId());
    sendFrame('typing.stop', { conversation_id: id });
  }

  function onTextChangeWithTyping(v: string) {
    setText(v);
    if (!id) return;
    if (v.trim()) {
      // Send typing.start (debounced: skip if sent within last 2s)
      const now = Date.now();
      if (now - lastTypingSent.current > 2000) {
        lastTypingSent.current = now;
        sendFrame('typing.start', { conversation_id: id });
      }
    } else {
      sendFrame('typing.stop', { conversation_id: id });
      lastTypingSent.current = 0;
    }
  }

  function onRetry(m: ChatItem) {
    if (m.status !== 'failed') return;
    const replyTo: ComposerReply | null = m.reply_to
      ? {
          id: m.reply_to.id,
          sender_id: m.reply_to.sender_id,
          body: m.reply_to.body,
          senderLabel: memberLabel(m.reply_to.sender_id),
        }
      : null;
    setMessages((prev) => prev.filter((x) => x.client_msg_id !== m.client_msg_id && x.id !== m.id));
    void doSend(m.body, replyTo, m.client_msg_id || newClientMsgId());
  }

  function onReply(m: ChatItem) {
    if (m.id.startsWith('local:') || m.status === 'pending' || m.status === 'failed') {
      return;
    }
    setReply({
      id: m.id,
      sender_id: m.sender_id,
      body: m.body,
      senderLabel: memberLabel(m.sender_id),
    });
  }

  async function onEditMessage(m: ChatItem, newBody: string) {
    if (!session.user || !id) return;
    try {
      const updated = await editMessage(id, m.id, newBody);
      setMessages((prev) => mergeMessage(prev, toChatItem(updated)));
    } catch (err) {
      setError(err instanceof ApiError ? err.message : t('common.requestFailed'));
    }
  }

  async function onRecallMessage(m: ChatItem) {
    if (!session.user || !id) return;
    try {
      const updated = await recallMessage(id, m.id);
      setMessages((prev) => mergeMessage(prev, toChatItem(updated)));
    } catch (err) {
      setError(err instanceof ApiError ? err.message : t('common.requestFailed'));
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

  const members = conv.members ?? [];
  const isGroup = members.length > 2;
  const peer = members.find((m) => m.id !== session.user?.id);
  // Live presence: override server-side initial value when a presence event arrives.
  const peerOnline = peer
    ? (presenceOverrides[peer.id] ?? peer.online ?? false)
    : false;

  // Compute min peer read seq for checkmark display
  const effectiveReadSeq = (() => {
    if (peerReadSeq.size === 0) return 0;
    if (!isGroup) {
      // DM: use peer's read seq directly
      return peer ? (peerReadSeq.get(peer.id) ?? 0) : 0;
    }
    // Group: min of all other members' read seqs
    let min = Infinity;
    for (const m of members) {
      if (m.id === session.user?.id) continue;
      const seq = peerReadSeq.get(m.id) ?? 0;
      if (seq < min) min = seq;
    }
    return min === Infinity ? 0 : min;
  })();

  const messagesWithRead = messages.map((m) =>
    m.sender_id === session.user?.id && m.status === 'sent' && m.seq <= effectiveReadSeq
      ? { ...m, isRead: true }
      : m,
  );

  // Typing indicator text
  const typingLabel = (() => {
    if (typingUsers.size === 0) return null;
    const names = [...typingUsers]
      .map((uid) => {
        const m = members.find((x) => x.id === uid);
        return m ? displayName(m.email, m.display_name) : uid.slice(0, 8);
      })
      .slice(0, 3);
    if (names.length === 1) return t('chat.typingOne', { name: names[0] });
    if (names.length === 2) return t('chat.typingTwo', { name: names[0], other: names[1] });
    return t('chat.typingMany');
  })();
  const explicitTitle = conv.title?.trim() ?? '';
  const title = explicitTitle
    ? explicitTitle
    : isGroup
      ? t('chat.groupUntitled')
      : peer
        ? displayName(peer.email, peer.display_name)
        : t('common.untitled');

  const isOwner = conv.created_by === session.user?.id;

  const runMemberAction = useCallback(
    async (action: 'leave' | 'kick' | 'transfer', targetId?: string) => {
      if (!id) return;
      setMemberNotice(null);
      try {
        if (action === 'leave') {
          await leaveGroup(id);
          navigate('/app');
          return;
        } else if (action === 'kick' && targetId) {
          await kickGroupMember(id, targetId);
        } else if (action === 'transfer' && targetId) {
          await transferGroupOwner(id, targetId);
        }
        // Re-fetch so the panel reflects membership/owner changes.
        const c = await getConversation(id);
        setConv(c);
      } catch (err) {
        setMemberNotice(err instanceof ApiError ? err.message : String(err));
      }
    },
    [id, navigate],
  );

  const startRename = useCallback(() => {
    setRenameValue(conv?.title ?? '');
    setMemberNotice(null);
    setRenaming(true);
  }, [conv]);

  const submitRename = useCallback(
    async (e: FormEvent) => {
      e.preventDefault();
      if (!id) return;
      const title = renameValue.trim();
      if (!title) {
        setMemberNotice(t('chat.renameBlank'));
        return;
      }
      setMemberNotice(null);
      try {
        const c = await renameGroup(id, title);
        setConv(c);
        setRenaming(false);
      } catch (err) {
        setMemberNotice(err instanceof ApiError ? err.message : String(err));
      }
    },
    [id, renameValue, t],
  );

  return (
    <section className="room">
      <header className="room__header">
        <div>
          <p className="page__eyebrow">{t('workspace.roomEyebrow')}</p>
          <h1 className="room__title">
            {!isGroup && (
              <span
                className="presence-dot presence-dot--inline"
                data-online={peerOnline ? 'true' : 'false'}
                aria-hidden
              />
            )}
            {title}
          </h1>
          {isGroup && (
            <p className="room__meta">
              {members.map((m) => m.email).join(' · ') || <code>{conv.id}</code>}
            </p>
          )}
        </div>
        {isGroup && (
          <button type="button" className="btn btn--ghost" onClick={() => setShowMembers((v) => !v)}>
            {t('chat.members')}
          </button>
        )}
      </header>

      {showMembers && isGroup && (
        <aside className="room__members" aria-label={t('chat.members')}>
          <h2 className="room__members-title">{t('chat.members')}</h2>
          <ul className="room__member-list">
            {members.map((m) => {
              const online = presenceOverrides[m.id] ?? m.online;
              return (
                <li key={m.id} className="room__member-row">
                  <span
                    className="presence-dot presence-dot--inline"
                    data-online={online ? 'true' : 'false'}
                    aria-hidden
                  />
                  {displayName(m.email, m.display_name)}
                  {m.id === conv.created_by && (
                    <span className="room__member-owner"> · {t('chat.owner')}</span>
                  )}
                  {isOwner && m.id !== session.user?.id && (
                    <span className="room__member-actions">
                      <button type="button" className="btn btn--ghost btn--sm" onClick={() => runMemberAction('transfer', m.id)}>
                        {t('chat.transferOwner')}
                      </button>
                      <button type="button" className="btn btn--ghost btn--sm" onClick={() => runMemberAction('kick', m.id)}>
                        {t('chat.kick')}
                      </button>
                    </span>
                  )}
                </li>
              );
            })}
          </ul>
          {memberNotice && (
            <p className="err" role="alert">
              {memberNotice}
            </p>
          )}
          {isOwner && renaming && (
            <form className="room__rename" onSubmit={submitRename}>
              <input
                className="field__input"
                type="text"
                maxLength={64}
                value={renameValue}
                onChange={(e) => setRenameValue(e.target.value)}
                placeholder={t('chat.renameGroupPlaceholder')}
                aria-label={t('chat.renameGroup')}
              />
              <span className="room__member-actions">
                <button type="submit" className="btn btn--ghost btn--sm">
                  {t('common.save')}
                </button>
                <button type="button" className="btn btn--ghost btn--sm" onClick={() => setRenaming(false)}>
                  {t('common.cancel')}
                </button>
              </span>
            </form>
          )}
          <div className="room__member-actions">
            {isOwner && !renaming && (
              <button type="button" className="btn btn--ghost btn--sm" onClick={startRename}>
                {t('chat.renameGroup')}
              </button>
            )}
            <button type="button" className="btn btn--ghost btn--sm" onClick={() => setShowAddMembers(true)}>
              {t('chat.addMembers')}
            </button>
            <button type="button" className="btn btn--ghost btn--sm" onClick={() => runMemberAction('leave')}>
              {t('chat.leaveGroup')}
            </button>
          </div>
        </aside>
      )}

      {showAddMembers && isGroup && conv && (
        <AddMembersDialog
          conversationId={conv.id}
          currentMemberIds={members.map((m) => m.id)}
          onClose={() => setShowAddMembers(false)}
          onAdded={() => void getConversation(conv.id).then(setConv).catch(() => undefined)}
        />
      )}

      <MessageList
        messages={messagesWithRead}
        selfId={session.user?.id}
        memberLabel={memberLabel}
        showSenderNames={isGroup}
        listRef={listRef}
        onReply={onReply}
        onRetry={onRetry}
        onEdit={onEditMessage}
        onRecall={onRecallMessage}
        emptyLabel={t('workspace.noMessages')}
      />

      {typingLabel && (
        <div className="typing-indicator" aria-live="polite">
          <span className="typing-indicator__dots">
            <span /><span /><span />
          </span>
          <span className="typing-indicator__text">{typingLabel}</span>
        </div>
      )}

      {error && (
        <p className="err room__err" role="alert">
          {error}
        </p>
      )}

      <Composer
        text={text}
        onTextChange={onTextChangeWithTyping}
        reply={reply}
        onClearReply={() => setReply(null)}
        sending={sending}
        onSend={onSend}
      />
    </section>
  );
}
