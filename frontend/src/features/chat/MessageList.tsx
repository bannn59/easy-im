import type { ReactNode, RefObject } from 'react';
import { MessageBubble, TimeDivider } from './MessageBubble';
import type { ChatItem } from './types';

const GAP_MS = 5 * 60 * 1000;

type Props = {
  messages: ChatItem[];
  selfId: string | undefined;
  memberLabel: (senderId: string) => string;
  /** Show per-bubble sender names (group chats only). */
  showSenderNames: boolean;
  listRef: RefObject<HTMLUListElement>;
  onReply: (m: ChatItem) => void;
  onRetry: (m: ChatItem) => void;
  onEdit: (m: ChatItem, newBody: string) => void;
  onRecall: (m: ChatItem) => void;
  emptyLabel: string;
  /** Message id to highlight (search jump target). */
  highlightId?: string;
};

export function MessageList({
  messages,
  selfId,
  memberLabel,
  showSenderNames,
  listRef,
  onReply,
  onRetry,
  onEdit,
  onRecall,
  emptyLabel,
  highlightId,
}: Props) {
  const nodes: ReactNode[] = [];
  let lastTs = 0;

  for (const m of messages) {
    const ts = new Date(m.created_at).getTime();
    if (!lastTs || (Number.isFinite(ts) && ts - lastTs > GAP_MS)) {
      nodes.push(<TimeDivider key={`t-${m.id}-${m.created_at}`} iso={m.created_at} />);
    }
    if (Number.isFinite(ts)) lastTs = ts;

    const mine = m.sender_id === selfId;
    const label = mine ? memberLabel(selfId ?? '') : memberLabel(m.sender_id);
    nodes.push(
      <MessageBubble
        key={m.localKey ?? m.id}
        message={m}
        mine={mine}
        senderLabel={label || m.sender_id}
        showSender={showSenderNames && !mine}
        resolveSender={memberLabel}
        onReply={onReply}
        onRetry={onRetry}
        onEdit={onEdit}
        onRecall={onRecall}
        highlight={m.id === highlightId}
      />,
    );
  }

  return (
    <ul className="msg-list" aria-live="polite" ref={listRef}>
      {messages.length === 0 ? <li className="muted msg-list__empty">{emptyLabel}</li> : nodes}
    </ul>
  );
}
