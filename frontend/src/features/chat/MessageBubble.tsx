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
};

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
}: Props) {
  const { t } = useTranslation();
  const status = message.status ?? 'sent';
  const name = shortName(senderLabel) || senderLabel;

  return (
    <li
      className={`bubble-item ${mine ? 'bubble-item--mine' : 'bubble-item--theirs'}${
        showSender ? ' bubble-item--named' : ''
      }`}
      data-status={status}
    >
      {showSender && !mine && <div className="bubble-sender">{name}</div>}
      <div className="bubble-row">
        {!mine && (
          <div className="bubble-avatar" aria-hidden>
            {initialsFrom(senderLabel)}
          </div>
        )}
        <div className="bubble-col">
          {message.reply_to && (
            <div className="bubble-quote">
              <span className="bubble-quote__who">
                {shortName(resolveSender(message.reply_to.sender_id)) ||
                  message.reply_to.sender_id.slice(0, 8)}
              </span>
              <span className="bubble-quote__body">{message.reply_to.body}</span>
            </div>
          )}
          <div className={`bubble ${mine ? 'bubble--mine' : 'bubble--theirs'}`}>
            <p className="bubble__body">{message.body}</p>
          </div>
          <div className="bubble-meta">
            <time dateTime={message.created_at}>{formatTime(message.created_at)}</time>
            {mine && status === 'sent' && (
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
            {status === 'sent' && (
              <button type="button" className="linkish bubble-reply-btn" onClick={() => onReply(message)}>
                {t('chat.reply')}
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
