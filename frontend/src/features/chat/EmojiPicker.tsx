import { useEffect, useRef } from 'react';
import { useTranslation } from 'react-i18next';
import { EMOJI_LIST } from './emoji';

type Props = {
  open: boolean;
  onClose: () => void;
  onPick: (emoji: string) => void;
};

export function EmojiPicker({ open, onClose, onPick }: Props) {
  const { t } = useTranslation();
  const rootRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    if (!open) return;
    function onDoc(ev: MouseEvent) {
      if (!rootRef.current) return;
      if (!rootRef.current.contains(ev.target as Node)) onClose();
    }
    document.addEventListener('mousedown', onDoc);
    return () => document.removeEventListener('mousedown', onDoc);
  }, [open, onClose]);

  if (!open) return null;

  return (
    <div className="emoji-picker" ref={rootRef} role="listbox" aria-label={t('chat.emoji')}>
      {EMOJI_LIST.map((e) => (
        <button
          key={e}
          type="button"
          className="emoji-picker__item"
          onClick={() => {
            onPick(e);
          }}
        >
          {e}
        </button>
      ))}
    </div>
  );
}
