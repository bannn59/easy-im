import { useState, type KeyboardEvent } from 'react';
import { useTranslation } from 'react-i18next';
import type { ChatItem } from './types';
import { initialsFrom, shortName } from './types';

type Props = {
  message: ChatItem;
  mine: boolean;
  senderLabel: string;
  /** Only true in group chats for others' messages. */
  showSender: boolean;
  resolveSender: (senderId: string) => string;
  onReply: (m: ChatItem) => void;
  onRetry?: (m: ChatItem) => void;
  onEdit?: (m: ChatItem, newBody: string) => void;
  onRecall?: (m: ChatItem) => void;
  /** Set when this message is the search jump target (visual highlight). */
  highlight?: boolean;
};

const EDIT_WINDOW_MS = 5 * 60 * 1000;

function formatTime(iso: string): string {
  const d = new Date(iso);
  if (Number.isNaN(d.getTime())) return '';
  return new Intl.DateTimeFormat(undefined, {
    hour: '2-digit',
    minute: '2-digit',
  }).format(d);
}

export function MessageBubble({
  message,
  mine,
  senderLabel,
  showSender,
  resolveSender,
  onReply,
  onRetry,
  onEdit,
  onRecall,
  highlight = false,
}: Props) {
  const { t } = useTranslation();
  const status = message.status ?? 'sent';
  const name = shortName(senderLabel) || senderLabel;
  const [editing, setEditing] = useState(false);
  const [draft, setDraft] = useState('');

  const recalled = Boolean(message.recalled_at);
  const canEditRecall =
    mine &&
    status === 'sent' &&
    !recalled &&
    Date.now() - new Date(message.created_at).getTime() <= EDIT_WINDOW_MS;

  function startEdit() {
    setDraft(message.body);
    setEditing(true);
  }

  function saveEdit(e?: KeyboardEvent) {
    e?.preventDefault();
    const trimmed = draft.trim();
    if (!trimmed) return;
    onEdit?.(message, trimmed);
    setEditing(false);
  }

  return (
    <li
      className={`bubble-item ${mine ? 'bubble-item--mine' : 'bubble-item--theirs'}${
        showSender ? ' bubble-item--named' : ''
      }${recalled ? ' bubble-item--recalled' : ''}${highlight ? ' bubble-item--highlight' : ''}`}
      data-status={status}
      data-message-id={message.id}
    >
      {showSender && !mine && <div className="bubble-sender">{name}</div>}
      <div className="bubble-row">
        {!mine && (
          <div className="bubble-avatar" aria-hidden>
            {initialsFrom(senderLabel)}
          </div>
        )}
        <div className="bubble-col">
          {message.reply_to && !recalled && (
            <div className="bubble-quote">
              <span className="bubble-quote__who">
                {shortName(resolveSender(message.reply_to.sender_id)) ||
                  message.reply_to.sender_id.slice(0, 8)}
              </span>
              <span className="bubble-quote__body">{message.reply_to.body}</span>
            </div>
          )}
          <div className={`bubble ${mine ? 'bubble--mine' : 'bubble--theirs'}`}>
            {recalled ? (
              <p className="bubble__body bubble__body--recalled">
                {t('chat.recalled')}
              </p>
            ) : editing ? (
              <textarea
                className="field__input composer__input bubble-edit-input"
                value={draft}
                onChange={(ev) => setDraft(ev.target.value)}
                onKeyDown={(ev) => {
                  if (ev.key === 'Enter' && !ev.shiftKey) saveEdit(ev);
                  if (ev.key === 'Escape') setEditing(false);
                }}
                rows={2}
                autoFocus
              />
            ) : (
              <p className="bubble__body">{message.body}</p>
            )}
          </div>
          <div className="bubble-meta">
            <time dateTime={message.created_at}>{formatTime(message.created_at)}</time>
            {message.edited_at && !recalled && <span>{t('chat.edited')}</span>}
            {mine && status === 'sent' && !recalled && (
              <span className="bubble-check" data-read={message.isRead ? 'true' : 'false'}>
                {message.isRead ? '✓✓' : '✓'}
              </span>
            )}
            {status === 'pending' && <span>{t('chat.sending')}</span>}
            {status === 'failed' && (
              <button type="button" className="linkish" onClick={() => onRetry?.(message)}>
                {t('chat.retry')}
              </button>
            )}
            {status === 'sent' && !recalled && !editing && (
              <>
                <button type="button" className="linkish bubble-reply-btn" onClick={() => onReply(message)}>
                  {t('chat.reply')}
                </button>
                {canEditRecall && onEdit && (
                  <button type="button" className="linkish bubble-reply-btn" onClick={startEdit}>
                    {t('chat.edit')}
                  </button>
                )}
                {canEditRecall && onRecall && (
                  <button type="button" className="linkish bubble-reply-btn" onClick={() => onRecall(message)}>
                    {t('chat.recall')}
                  </button>
                )}
              </>
            )}
            {editing && (
              <button type="button" className="linkish bubble-reply-btn" onClick={() => setEditing(false)}>
                {t('chat.cancelReply')}
              </button>
            )}
          </div>
        </div>
        {mine && (
          <div className="bubble-avatar bubble-avatar--mine" aria-hidden>
            {initialsFrom(senderLabel)}
          </div>
        )}
      </div>
    </li>
  );
}

type DividerProps = { iso: string };

export function TimeDivider({ iso }: DividerProps) {
  const d = new Date(iso);
  const label = Number.isNaN(d.getTime())
    ? iso
    : new Intl.DateTimeFormat(undefined, {
        month: 'short',
        day: 'numeric',
        hour: '2-digit',
        minute: '2-digit',
      }).format(d);
  return (
    <li className="msg-time-divider" aria-hidden>
      <span>{label}</span>
    </li>
  );
}
