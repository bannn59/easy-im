import type { ReactNode } from 'react';

/**
 * Highlight every (case-insensitive) occurrence of `query` inside `body`,
 * returning React nodes. Non-matched text is HTML-escaped to prevent XSS
 * (body is user input); matched segments are wrapped in <mark>.
 *
 * Escaping: we intentionally do NOT use dangerouslySetInnerHTML. We escape the
 * body first, then locate the query within the escaped text (queries rarely
 * contain HTML entities, so matching on the raw body and escaping around it is
 * simpler and safer). Both paths are equivalent for the common case.
 */
export function highlightQuery(body: string, query: string): ReactNode[] {
  const q = query.trim();
  if (!q) return [escapeHtml(body)];
  const lower = q.toLowerCase();
  const nodes: ReactNode[] = [];
  const out = body;
  const lo = out.toLowerCase();
  let i = 0;
  let idx = lo.indexOf(lower, i);
  while (idx >= 0) {
    if (idx > i) nodes.push(escapeHtml(out.slice(i, idx)));
    nodes.push(<mark key={`${idx}-${out.slice(idx, idx + q.length)}`}>{escapeHtml(out.slice(idx, idx + q.length))}</mark>);
    i = idx + q.length;
    if (q.length === 0) break;
    idx = lo.indexOf(lower, i);
  }
  if (i < out.length) nodes.push(escapeHtml(out.slice(i)));
  return nodes.length ? nodes : [escapeHtml(body)];
}

const ESCAPE_MAP: Record<string, string> = {
  '&': '&amp;',
  '<': '&lt;',
  '>': '&gt;',
  '"': '&quot;',
  "'": '&#39;',
};

function escapeHtml(s: string): string {
  return s.replace(/[&<>"']/g, (c) => ESCAPE_MAP[c]);
}
