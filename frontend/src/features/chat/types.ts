export type ChatStatus = 'pending' | 'sent' | 'failed';

export type ChatItem = {
  id: string;
  conversation_id: string;
  sender_id: string;
  body: string;
  client_msg_id: string;
  seq: number;
  created_at: string;
  reply_to?: {
    id: string;
    sender_id: string;
    body: string;
  } | null;
  status?: ChatStatus;
  localKey?: string;
  isRead?: boolean;
};

export function newClientMsgId(): string {
  if (typeof crypto !== 'undefined' && 'randomUUID' in crypto) {
    return crypto.randomUUID();
  }
  return `c-${Date.now()}-${Math.random().toString(16).slice(2)}`;
}

export function mergeMessage(prev: ChatItem[], incoming: ChatItem): ChatItem[] {
  const idx = prev.findIndex(
    (x) =>
      x.id === incoming.id ||
      (incoming.client_msg_id && x.client_msg_id === incoming.client_msg_id) ||
      (incoming.localKey && x.localKey === incoming.localKey),
  );
  if (idx >= 0) {
    const next = prev.slice();
    next[idx] = { ...prev[idx], ...incoming, status: incoming.status ?? 'sent' };
    return next.sort((a, b) => a.seq - b.seq || a.created_at.localeCompare(b.created_at));
  }
  return [...prev, { ...incoming, status: incoming.status ?? 'sent' }].sort(
    (a, b) => a.seq - b.seq || a.created_at.localeCompare(b.created_at),
  );
}

export function initialsFrom(label: string): string {
  const s = label.trim();
  if (!s) return '?';
  const local = s.includes('@') ? s.split('@')[0] : s;
  return local.slice(0, 1).toUpperCase();
}

export function shortName(label: string): string {
  const s = label.trim();
  if (!s) return '';
  if (s.includes('@')) return s.split('@')[0];
  return s;
}

/** Insert text at selection inside a controlled string. */
export function insertAtCursor(
  value: string,
  insert: string,
  start: number,
  end: number,
): { value: string; caret: number } {
  const s = Math.max(0, Math.min(start, value.length));
  const e = Math.max(s, Math.min(end, value.length));
  const next = value.slice(0, s) + insert + value.slice(e);
  return { value: next, caret: s + insert.length };
}
