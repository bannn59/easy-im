import { useCallback, useEffect, useRef, useState } from 'react';
import { useParams } from 'react-router-dom';
import { useTranslation } from 'react-i18next';
import { getConversation, markConversationRead, type Conversation } from '../../api/conversations';
import { editMessage, listMessages, recallMessage, sendMessage, type Message } from '../../api/messages';
import { ApiError } from '../../api/http';
import { useRealtime, sendFrame } from '../../realtime';
import { useSession } from '../../app/Session';
import { Composer, type ComposerReply } from './Composer';
import { MessageList } from './MessageList';
import { displayName, mergeMessage, newClientMsgId, shortName, type ChatItem } from './types';

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
    if (!session.token || !id) return;
    const res = await listMessages(session.token, id, { limit: 100 });
    setMessages((res.messages ?? []).map((m) => toChatItem(m)));
  }, [session.token, id]);

  useEffect(() => {
    if (!session.token || !id) return;
    let cancelled = false;
    setLoading(true);
    setError(null);
    setReply(null);
    setText('');
    stickToBottom.current = true;
    Promise.all([getConversation(session.token, id), listMessages(session.token, id, { limit: 100 })])
      .then(async ([c, msgRes]) => {
        if (cancelled) return;
        setConv(c);
        setMessages((msgRes.messages ?? []).map((m) => toChatItem(m)));
        // Self-only unread: opening the room marks read to head.
        try {
          await markConversationRead(session.token!, id);
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
  }, [id, session.token, t]);

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
  const token = session.token;
  useRealtime({
    onMessageCreated: (m) => {
      if (m.conversation_id !== convId || !token) return;
      setMessages((prev) => mergeMessage(prev, toChatItem(m)));
      requestAnimationFrame(() => scrollToBottom(false));
      void markConversationRead(token, convId, m.seq).catch(() => undefined);
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
  });

  // 15s polling fallback while the room is open.
  useEffect(() => {
    if (!session.token || !id) return;
    const timer = window.setInterval(() => {
      void loadMessages().catch(() => undefined);
    }, 15000);
    return () => window.clearInterval(timer);
  }, [session.token, id, loadMessages]);

  async function doSend(body: string, replyTo: ComposerReply | null, clientMsgId: string) {
    if (!session.token || !id || !session.user) return;
    const trimmed = body.trim();
    if (!trimmed) return;

    const token = session.token;
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
      const m = await sendMessage(token, convId, {
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
    if (!session.token || !id) return;
    try {
      const updated = await editMessage(session.token, id, m.id, newBody);
      setMessages((prev) => mergeMessage(prev, toChatItem(updated)));
    } catch (err) {
      setError(err instanceof ApiError ? err.message : t('common.requestFailed'));
    }
  }

  async function onRecallMessage(m: ChatItem) {
    if (!session.token || !id) return;
    try {
      const updated = await recallMessage(session.token, id, m.id);
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
      </header>

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
