import { useRef, useState, type FormEvent, type KeyboardEvent } from 'react';
import { useTranslation } from 'react-i18next';
import { EmojiPicker } from './EmojiPicker';
import { ReplyBar } from './ReplyBar';
import { insertAtCursor } from './types';

export type ComposerReply = {
  id: string;
  sender_id: string;
  body: string;
  senderLabel?: string;
};

type Props = {
  text: string;
  onTextChange: (v: string) => void;
  reply: ComposerReply | null;
  onClearReply: () => void;
  sending: boolean;
  onSend: () => void;
};

export function Composer({ text, onTextChange, reply, onClearReply, sending, onSend }: Props) {
  const { t } = useTranslation();
  const [emojiOpen, setEmojiOpen] = useState(false);
  const taRef = useRef<HTMLTextAreaElement>(null);

  function submit(e?: FormEvent) {
    e?.preventDefault();
    if (!text.trim() || sending) return;
    onSend();
    setEmojiOpen(false);
  }

  function onKeyDown(ev: KeyboardEvent<HTMLTextAreaElement>) {
    if (ev.key === 'Enter' && !ev.shiftKey) {
      ev.preventDefault();
      submit();
    }
  }

  function pickEmoji(emoji: string) {
    const el = taRef.current;
    const start = el?.selectionStart ?? text.length;
    const end = el?.selectionEnd ?? text.length;
    const { value, caret } = insertAtCursor(text, emoji, start, end);
    onTextChange(value);
    requestAnimationFrame(() => {
      const node = taRef.current;
      if (!node) return;
      node.focus();
      node.setSelectionRange(caret, caret);
    });
  }

  return (
    <form className="composer" onSubmit={submit}>
      {reply && <ReplyBar target={reply} onCancel={onClearReply} />}
      <div className="composer__tools">
        <div className="composer__emoji-wrap">
          <button
            type="button"
            className="btn btn--ghost composer__tool"
            aria-expanded={emojiOpen}
            aria-label={t('chat.emoji')}
            onClick={() => setEmojiOpen((o) => !o)}
          >
            😊
          </button>
          <EmojiPicker open={emojiOpen} onClose={() => setEmojiOpen(false)} onPick={pickEmoji} />
        </div>
      </div>
      <label className="field__label visually-hidden" htmlFor="msg-body">
        {t('workspace.messageLabel')}
      </label>
      <textarea
        id="msg-body"
        ref={taRef}
        className="field__input composer__input"
        value={text}
        onChange={(ev) => onTextChange(ev.target.value)}
        onKeyDown={onKeyDown}
        placeholder={t('workspace.messagePlaceholder')}
        rows={2}
        autoComplete="off"
      />
      <div className="composer__actions">
        <span className="muted composer__hint">{t('chat.composerHint')}</span>
        <button className="btn" type="submit" disabled={sending || !text.trim()}>
          {sending ? t('workspace.sending') : t('workspace.send')}
        </button>
      </div>
    </form>
  );
}
