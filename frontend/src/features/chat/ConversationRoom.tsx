import { useCallback, useEffect, useRef, useState } from 'react';
import { useParams } from 'react-router-dom';
import { useTranslation } from 'react-i18next';
import { getConversation, type Conversation } from '../../api/conversations';
import { listMessages, sendMessage, type Message } from '../../api/messages';
import { ApiError } from '../../api/http';
import { connectRealtime } from '../../realtime';
import { useSession } from '../../app/Session';
import { Composer, type ComposerReply } from './Composer';
import { MessageList } from './MessageList';
import { mergeMessage, newClientMsgId, shortName, type ChatItem } from './types';

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
  const listRef = useRef<HTMLUListElement>(null!);
  const stickToBottom = useRef(true);

  const memberLabel = useCallback(
    (senderId: string) => {
      if (senderId && senderId === session.user?.id) {
        return session.user.email;
      }
      const m = conv?.members?.find((x) => x.id === senderId);
      return m?.email ?? senderId.slice(0, 8);
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
      .then(([c, msgRes]) => {
        if (!cancelled) {
          setConv(c);
          setMessages((msgRes.messages ?? []).map((m) => toChatItem(m)));
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

  useEffect(() => {
    if (!session.token || !id || !conv) return;
    const stop = connectRealtime(session.token, {
      onMessageCreated: (m) => {
        if (m.conversation_id !== id) return;
        setMessages((prev) => {
          const next = mergeMessage(prev, toChatItem(m));
          return next;
        });
        requestAnimationFrame(() => scrollToBottom(false));
      },
    });
    const timer = window.setInterval(() => {
      void loadMessages().catch(() => undefined);
    }, 15000);
    return () => {
      stop();
      window.clearInterval(timer);
    };
  }, [session.token, id, conv, loadMessages, scrollToBottom]);

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
  const explicitTitle = conv.title?.trim() ?? '';
  const title = explicitTitle
    ? explicitTitle
    : isGroup
      ? t('chat.groupUntitled')
      : peer
        ? shortName(peer.email) || peer.email
        : t('common.untitled');

  return (
    <section className="room">
      <header className="room__header">
        <div>
          <p className="page__eyebrow">{t('workspace.roomEyebrow')}</p>
          <h1 className="room__title">{title}</h1>
          {isGroup && (
            <p className="room__meta">
              {members.map((m) => m.email).join(' · ') || <code>{conv.id}</code>}
            </p>
          )}
        </div>
      </header>

      <MessageList
        messages={messages}
        selfId={session.user?.id}
        memberLabel={memberLabel}
        showSenderNames={isGroup}
        listRef={listRef}
        onReply={onReply}
        onRetry={onRetry}
        emptyLabel={t('workspace.noMessages')}
      />

      {error && (
        <p className="err room__err" role="alert">
          {error}
        </p>
      )}

      <Composer
        text={text}
        onTextChange={setText}
        reply={reply}
        onClearReply={() => setReply(null)}
        sending={sending}
        onSend={onSend}
      />
    </section>
  );
}
