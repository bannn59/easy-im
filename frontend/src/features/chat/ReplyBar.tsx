import { useTranslation } from 'react-i18next';
import { shortName } from './types';

type ReplyTarget = {
  id: string;
  sender_id: string;
  body: string;
  senderLabel?: string;
};

type Props = {
  target: ReplyTarget;
  onCancel: () => void;
};

export function ReplyBar({ target, onCancel }: Props) {
  const { t } = useTranslation();
  const who = target.senderLabel ? shortName(target.senderLabel) : shortName(target.sender_id);
  return (
    <div className="reply-bar" role="status">
      <div className="reply-bar__main">
        <span className="reply-bar__label">{t('chat.replyingTo', { name: who || target.sender_id })}</span>
        <span className="reply-bar__body">{target.body}</span>
      </div>
      <button type="button" className="btn btn--ghost reply-bar__cancel" onClick={onCancel}>
        {t('chat.cancelReply')}
      </button>
    </div>
  );
}
