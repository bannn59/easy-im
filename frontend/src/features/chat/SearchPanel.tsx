import { useCallback, useEffect, useState, type FormEvent } from 'react';
import { useTranslation } from 'react-i18next';
import { searchMessages, type Message } from '../../api/messages';
import { ApiError } from '../../api/http';
import { displayName } from './types';
import type { PublicUser } from '../../api/auth';

const PAGE = 50;

/** Panel to search messages in the current conversation and jump to a result. */
export default function SearchPanel({
  conversationId,
  members,
  selfId,
  onJump,
  onClose,
}: {
  conversationId: string;
  members: PublicUser[];
  selfId: string | undefined;
  onJump: (seq: number, messageId: string) => void;
  onClose: () => void;
}) {
  const { t } = useTranslation();
  const [query, setQuery] = useState('');
  const [results, setResults] = useState<Message[] | null>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [searched, setSearched] = useState('');

  const resolveLabel = useCallback(
    (m: Message) => {
      if (m.sender_id === selfId) return t('common.you');
      const peer = members.find((x) => x.id === m.sender_id);
      return peer ? displayName(peer.email, peer.display_name) : m.sender_id.slice(0, 8);
    },
    [members, selfId, t],
  );

  // Reset state when switching conversations.
  useEffect(() => {
    setQuery('');
    setResults(null);
    setError(null);
    setSearched('');
  }, [conversationId]);

  async function runSearch(beforeSeq: number) {
    if (!query.trim()) return;
    setLoading(true);
    setError(null);
    try {
      const res = await searchMessages(conversationId, query.trim(), {
        before_seq: beforeSeq || undefined,
        limit: PAGE,
      });
      const msgs = res.messages ?? [];
      setResults((prev) => {
        const seen = new Set((prev ?? []).map((m) => m.id));
        return [...(prev ?? []), ...msgs.filter((m) => !seen.has(m.id))];
      });
      setSearched(query.trim());
    } catch (err) {
      setError(err instanceof ApiError ? err.message : t('common.failedToLoad'));
    } finally {
      setLoading(false);
    }
  }

  function onSubmit(e: FormEvent) {
    e.preventDefault();
    if (!query.trim()) return;
    setResults(null);
    void runSearch(0);
  }

  return (
    <aside className="room__search" aria-label={t('chat.searchTitle')}>
      <form className="room__search-form" onSubmit={onSubmit}>
        <input
          className="field__input"
          type="search"
          value={query}
          onChange={(e) => setQuery(e.target.value)}
          placeholder={t('chat.searchPlaceholder')}
          aria-label={t('chat.searchTitle')}
          autoFocus
        />
        <span className="room__member-actions">
          <button type="submit" className="btn btn--ghost btn--sm" disabled={loading || !query.trim()}>
            {loading ? t('common.loading') : t('chat.search')}
          </button>
          <button type="button" className="btn btn--ghost btn--sm" onClick={onClose}>
            {t('common.cancel')}
          </button>
        </span>
      </form>

      {error && (
        <p className="err" role="alert">
          {error}
        </p>
      )}

      {results !== null && results.length === 0 && (
        <p className="muted room__search-empty">{t('chat.searchEmpty', { q: searched })}</p>
      )}

      {results !== null && results.length > 0 && (
        <ul className="room__search-results">
          {results.map((m) => (
            <li key={m.id}>
              <button
                type="button"
                className="room__search-result"
                onClick={() => onJump(m.seq, m.id)}
              >
                <span className="room__search-result-who">{resolveLabel(m)}</span>
                <span className="room__search-result-body">{m.body}</span>
              </button>
            </li>
          ))}
        </ul>
      )}

      {results !== null && results.length >= PAGE && (
        <div className="room__member-actions">
          <button type="button" className="btn btn--ghost btn--sm" onClick={() => void runSearch(results[results.length - 1].seq)} disabled={loading}>
            {t('chat.searchMore')}
          </button>
        </div>
      )}
    </aside>
  );
}
