import { useEffect, useRef } from 'react';
import { getApiBase } from '../api/http';
import type { Message } from '../api/messages';
import { useSession } from '../app/Session';

export type RealtimeStatus = 'idle' | 'connecting' | 'connected' | 'offline';

export type ReadReceiptData = {
  conversation_id: string;
  reader_id: string;
  last_read_seq: number;
};

export type TypingData = {
  conversation_id: string;
  user_id: string;
};

export type PresenceData = {
  user_id: string;
  online: boolean;
};

export type MembersChangedData = {
  conversation_id: string;
  action: 'added' | 'left' | 'kicked' | 'owner_transferred';
  user_id: string;
  members: string[];
};

export type ConversationRenamedData = {
  conversation_id: string;
  title: string;
  updated_at: string;
};

export type RealtimeHandlers = {
  onMessageCreated?: (m: Message) => void;
  onMessageEdited?: (m: Message) => void;
  onMessageRecalled?: (m: Message) => void;
  onMessageRead?: (data: ReadReceiptData) => void;
  onTypingStarted?: (data: TypingData) => void;
  onTypingStopped?: (data: TypingData) => void;
  onPresenceChanged?: (data: PresenceData) => void;
  onMembersChanged?: (data: MembersChangedData) => void;
  onConversationRenamed?: (data: ConversationRenamedData) => void;
  onStatus?: (s: RealtimeStatus) => void;
};

type IncomingFrame = { type?: string; payload?: unknown };

// ---- Singleton connection (established by RealtimeProvider) ----
let singletonWs: WebSocket | null = null;
let connected = false;
let retry = 0;
let retryTimer: number | undefined;
let stopping = false;
let status: RealtimeStatus = 'idle';

const subscribers = new Set<RealtimeHandlers>();
const statusListeners = new Set<(s: RealtimeStatus) => void>();

function setStatus(s: RealtimeStatus) {
  if (status === s) return;
  status = s;
  for (const fn of statusListeners) fn(s);
  for (const h of subscribers) h.onStatus?.(s);
}

function notify(apply: (h: RealtimeHandlers) => void) {
  for (const h of subscribers) apply(h);
}

function dispatch(frame: IncomingFrame) {
  if (!frame.type || !frame.payload) return;
  switch (frame.type) {
    case 'message.created':
      notify((h) => h.onMessageCreated?.(frame.payload as Message));
      break;
    case 'message.edited':
      notify((h) => h.onMessageEdited?.(frame.payload as Message));
      break;
    case 'message.recalled':
      notify((h) => h.onMessageRecalled?.(frame.payload as Message));
      break;
    case 'message.read':
      notify((h) => h.onMessageRead?.(frame.payload as ReadReceiptData));
      break;
    case 'typing.started':
      notify((h) => h.onTypingStarted?.(frame.payload as TypingData));
      break;
    case 'typing.stopped':
      notify((h) => h.onTypingStopped?.(frame.payload as TypingData));
      break;
    case 'presence.changed':
      notify((h) => h.onPresenceChanged?.(frame.payload as PresenceData));
      break;
    case 'members.changed':
      notify((h) => h.onMembersChanged?.(frame.payload as MembersChangedData));
      break;
    case 'conversation.renamed':
      notify((h) => h.onConversationRenamed?.(frame.payload as ConversationRenamedData));
      break;
  }
}

function open() {
  if (stopping) return;
  setStatus('connecting');
  const base = getApiBase().replace(/^http/, 'ws');
  const ws = new WebSocket(`${base}/v1/ws`); // cookie-authenticated; no ?token=
  singletonWs = ws;
  ws.onopen = () => {
    retry = 0;
    setStatus('connected');
  };
  ws.onmessage = (ev) => {
    try {
      dispatch(JSON.parse(String(ev.data)) as IncomingFrame);
    } catch {
      // ignore malformed
    }
  };
  ws.onclose = () => {
    if (singletonWs === ws) singletonWs = null;
    setStatus('offline');
    if (!stopping) {
      const delay = Math.min(10000, 500 * 2 ** retry);
      retry += 1;
      retryTimer = window.setTimeout(open, delay);
    }
  };
  ws.onerror = () => {
    ws.close();
  };
}

/**
 * Establish the app-wide WebSocket connection. Idempotent; a no-op if the
 * connection (or its retry timer) is already active.
 */
export function startRealtime(): void {
  if (connected || retryTimer !== undefined) {
    return;
  }
  stopping = false;
  open();
}

/** Close the app-wide WebSocket connection (e.g. on logout). */
export function stopRealtime(): void {
  stopping = true;
  if (retryTimer !== undefined) {
    window.clearTimeout(retryTimer);
    retryTimer = undefined;
  }
  if (singletonWs) {
    singletonWs.close();
    singletonWs = null;
  }
  connected = false;
  retry = 0;
  setStatus('idle');
}

/** Subscribe handlers to the shared connection. Returns an unsubscribe fn. */
export function subscribeRealtime(handlers: RealtimeHandlers): () => void {
  subscribers.add(handlers);
  return () => {
    subscribers.delete(handlers);
  };
}

/** React hook: subscribe handlers to the shared connection for this mount. */
export function useRealtime(handlers: RealtimeHandlers): void {
  const ref = useRef(handlers);
  ref.current = handlers;
  useEffect(() => {
    const proxy: RealtimeHandlers = {
      onMessageCreated: (m) => ref.current.onMessageCreated?.(m),
      onMessageEdited: (m) => ref.current.onMessageEdited?.(m),
      onMessageRecalled: (m) => ref.current.onMessageRecalled?.(m),
      onMessageRead: (d) => ref.current.onMessageRead?.(d),
      onTypingStarted: (d) => ref.current.onTypingStarted?.(d),
      onTypingStopped: (d) => ref.current.onTypingStopped?.(d),
      onPresenceChanged: (d) => ref.current.onPresenceChanged?.(d),
      onMembersChanged: (d) => ref.current.onMembersChanged?.(d),
      onConversationRenamed: (d) => ref.current.onConversationRenamed?.(d),
      onStatus: (s) => ref.current.onStatus?.(s),
    };
    return subscribeRealtime(proxy);
  }, []);
}

/** Send a frame to the server over the shared connection. */
export function sendFrame(type: string, payload: unknown): void {
  if (!singletonWs || singletonWs.readyState !== WebSocket.OPEN) return;
  singletonWs.send(JSON.stringify({ type, payload }));
}

/**
 * App-level realtime connection lifecycle. Renders children and keeps the
 * singleton WebSocket connected while a user is logged in.
 */
export function RealtimeProvider({ children }: { children: React.ReactNode }) {
  const session = useSession();
  useEffect(() => {
    if (session.user) {
      startRealtime();
      return () => stopRealtime();
    }
    stopRealtime();
  }, [session.user]);
  return <>{children}</>;
}
