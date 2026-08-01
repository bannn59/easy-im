import { useCallback, useEffect, useState, type FormEvent } from 'react';
import { useTranslation } from 'react-i18next';
import { listFriends } from '../../api/friends';
import { addGroupMembers } from '../../api/conversations';
import type { PublicUser } from '../../api/auth';
import { ApiError } from '../../api/http';
import { displayName } from './types';

/** Modal for adding friend members to an existing group. */
export default function AddMembersDialog({
  conversationId,
  currentMemberIds,
  onClose,
  onAdded,
}: {
  conversationId: string;
  currentMemberIds: string[];
  onClose: () => void;
  onAdded?: () => void;
}) {
  const { t } = useTranslation();
  const [friends, setFriends] = useState<PublicUser[] | null>(null);
  const [selected, setSelected] = useState<Set<string>>(new Set());
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const existing = new Set(currentMemberIds);

  useEffect(() => {
    listFriends()
      .then((res) => {
        const friendList = (res.friends ?? []).filter((f) => !existing.has(f.id));
        setFriends(friendList);
      })
      .catch(() => setFriends([]));
  }, []); // eslint-disable-line react-hooks/exhaustive-deps

  const toggle = useCallback((id: string) => {
    setSelected((prev) => {
      const next = new Set(prev);
      if (next.has(id)) {
        next.delete(id);
      } else {
        next.add(id);
      }
      return next;
    });
  }, []);

  const onSubmit = useCallback(
    async (e: FormEvent) => {
      e.preventDefault();
      if (selected.size === 0) {
        setError(t('chat.noMembers'));
        return;
      }
      setLoading(true);
      setError(null);
      try {
        await addGroupMembers(conversationId, [...selected]);
        onAdded?.();
        onClose();
      } catch (err) {
        setError(err instanceof ApiError ? err.message : String(err));
        setLoading(false);
      }
    },
    [selected, conversationId, onClose, onAdded, t],
  );

  return (
    <div className="modal-overlay" onClick={onClose}>
      <form className="modal" onSubmit={onSubmit} onClick={(e) => e.stopPropagation()}>
        <h2 className="modal__title">{t('chat.addMembers')}</h2>

        <fieldset className="groupchat__members">
          <legend className="field__label">{t('chat.pickMembers')}</legend>
          {friends === null && <p className="err">{t('common.failedToLoad')}</p>}
          {friends !== null && friends.length === 0 && <p className="muted">{t('chat.noAddableFriends')}</p>}
          {friends !== null && (
            <ul className="groupchat__member-list">
              {friends.map((f) => (
                <li key={f.id}>
                  <label className="groupchat__member-row">
                    <input
                      type="checkbox"
                      checked={selected.has(f.id)}
                      onChange={() => toggle(f.id)}
                    />
                    <span>{displayName(f.email, f.display_name)}</span>
                  </label>
                </li>
              ))}
            </ul>
          )}
        </fieldset>

        {error && (
          <p className="err" role="alert">
            {error}
          </p>
        )}

        <div className="modal__actions">
          <button type="button" className="btn btn--ghost" onClick={onClose}>
            {t('common.cancel')}
          </button>
          <button type="submit" className="btn" disabled={loading}>
            {loading ? t('common.loading') : t('chat.addMembers')}
          </button>
        </div>
      </form>
    </div>
  );
}
