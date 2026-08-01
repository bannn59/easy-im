import { useCallback, useState, type FormEvent } from 'react';
import { useNavigate } from 'react-router-dom';
import { useTranslation } from 'react-i18next';
import { globalSearchMessages, type GlobalSearchResult } from '../../api/messages';
import { ApiError } from '../../api/http';
import { useSession } from '../../app/Session';
import { highlightQuery } from './searchHighlight';

const PAGE = 50;

/**
 * Global search across all the user's conversations. Results show the
 * conversation title plus a highlighted snippet; clicking navigates to that
 * conversation.
 */
export default function GlobalSearchPanel({ onClose }: { onClose?: () => void }) {
  const { t } = useTranslation();
  const navigate = useNavigate();
  const session = useSession();
  const [query, setQuery] = useState('');
  const [results, setResults] = useState<GlobalSearchResult[] | null>(null);
  const [nextCursor, setNextCursor] = useState<string | null>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const selfId = session.user?.id;

  const titleFor = useCallback(
    (r: GlobalSearchResult) => {
      if (r.conversation_title?.trim()) return r.conversation_title.trim();
      // DM with null title: fall back to a generic label.
      return r.sender_id === selfId ? t('chat.youSaid') : t('chat.conversation');
    },
    [selfId, t],
  );

  const whoFor = useCallback(
    (r: GlobalSearchResult) => {
      if (r.sender_id === selfId) return t('common.you');
      return r.sender_id.slice(0, 8);
    },
    [selfId, t],
  );

  async function runSearch(cursor: string | null) {
    if (!query.trim()) return;
    setLoading(true);
    setError(null);
    try {
      const res = await globalSearchMessages(query.trim(), {
        cursor: cursor ?? undefined,
        limit: PAGE,
      });
      const msgs = res.messages ?? [];
      setResults((prev) => {
        const seen = new Set((prev ?? []).map((m) => m.id));
        return [...(prev ?? []), ...msgs.filter((m) => !seen.has(m.id))];
      });
      setNextCursor(res.next_cursor || null);
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
    setNextCursor(null);
    void runSearch(null);
  }

  function jumpTo(r: GlobalSearchResult) {
    navigate(`/app/c/${r.conversation_id}`);
    onClose?.();
  }

  return (
    <section className="page">
      <p className="page__eyebrow">{t('chat.globalSearchTitle')}</p>
      <h1 className="page__title">{t('chat.globalSearchTitle')}</h1>

      <form className="friends__send" onSubmit={onSubmit}>
        <div className="field">
          <label className="field__label" htmlFor="global-search-q">
            {t('chat.globalSearchPlaceholder')}
          </label>
          <input
            id="global-search-q"
            className="field__input"
            type="search"
            value={query}
            onChange={(e) => setQuery(e.target.value)}
            placeholder={t('chat.globalSearchPlaceholder')}
            autoFocus
          />
        </div>
        <button className="btn" type="submit" disabled={loading || !query.trim()}>
          {loading ? t('common.loading') : t('chat.search')}
        </button>
        <button type="button" className="btn btn--ghost" onClick={onClose}>
          {t('common.cancel')}
        </button>
      </form>

      {error && (
        <p className="err" role="alert">
          {error}
        </p>
      )}

      {results !== null && results.length === 0 && (
        <p className="muted">{t('chat.searchEmpty', { q: query.trim() })}</p>
      )}

      {results !== null && results.length > 0 && (
        <ul className="global-search__list">
          {results.map((r) => (
            <li key={r.id}>
              <button type="button" className="global-search__result" onClick={() => jumpTo(r)}>
                <span className="global-search__conv">{titleFor(r)}</span>
                <span className="global-search__body">{highlightQuery(r.body, query)}</span>
                <span className="global-search__who muted">{whoFor(r)}</span>
              </button>
            </li>
          ))}
        </ul>
      )}

      {nextCursor && (
        <div className="page__actions">
          <button type="button" className="btn btn--ghost" onClick={() => void runSearch(nextCursor)} disabled={loading}>
            {loading ? t('common.loading') : t('chat.searchMore')}
          </button>
        </div>
      )}
    </section>
  );
}
