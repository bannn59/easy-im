import { useCallback, useEffect, useState, type FormEvent } from 'react';
import { useNavigate } from 'react-router-dom';
import { useTranslation } from 'react-i18next';
import { listFriends } from '../../api/friends';
import { createGroup } from '../../api/conversations';
import type { PublicUser } from '../../api/auth';
import { ApiError } from '../../api/http';
import { displayName } from './types';

/** Modal for creating a group chat: pick friends + optional name. */
export default function CreateGroupDialog({
  onClose,
  onCreated,
}: {
  onClose: () => void;
  onCreated?: () => void;
}) {
  const { t } = useTranslation();
  const navigate = useNavigate();
  const [friends, setFriends] = useState<PublicUser[] | null>(null);
  const [selected, setSelected] = useState<Set<string>>(new Set());
  const [title, setTitle] = useState('');
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    listFriends()
      .then((res) => setFriends(res.friends ?? []))
      .catch(() => setFriends([]));
  }, []);

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
        const conv = await createGroup(title.trim(), [...selected]);
        onCreated?.();
        onClose();
        navigate(`/app/c/${conv.id}`);
      } catch (err) {
        setError(err instanceof ApiError ? err.message : String(err));
        setLoading(false);
      }
    },
    [selected, title, navigate, onClose, t],
  );

  return (
    <div className="modal-overlay" onClick={onClose}>
      <form className="modal" onSubmit={onSubmit} onClick={(e) => e.stopPropagation()}>
        <h2 className="modal__title">{t('chat.create')}</h2>

        <label className="field__label" htmlFor="group-title">
          {t('chat.nameLabel')}
        </label>
        <input
          id="group-title"
          className="field__input"
          value={title}
          onChange={(e) => setTitle(e.target.value)}
          placeholder={t('chat.namePlaceholder')}
        />

        <fieldset className="groupchat__members">
          <legend className="field__label">{t('chat.pickMembers')}</legend>
          {friends === null && <p className="err">{t('common.failedToLoad')}</p>}
          {friends !== null && friends.length === 0 && <p className="muted">{t('chat.noFriends')}</p>}
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
            {loading ? t('common.loading') : t('chat.create')}
          </button>
        </div>
      </form>
    </div>
  );
}
