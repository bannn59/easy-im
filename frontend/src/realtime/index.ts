import { getApiBase } from '../api/http';
import type { Message } from '../api/messages';

export type RealtimeStatus = 'idle' | 'connecting' | 'connected' | 'offline';

export type RealtimeHandlers = {
  onMessageCreated?: (m: Message) => void;
  onMessageRead?: (data: { conversation_id: string; reader_id: string; last_read_seq: number }) => void;
  onTypingStarted?: (data: { conversation_id: string; user_id: string }) => void;
  onTypingStopped?: (data: { conversation_id: string; user_id: string }) => void;
  onStatus?: (s: RealtimeStatus) => void;
};

let activeWs: WebSocket | null = null;

/**
 * Opens a websocket to /v1/ws?token=...
 * Do not construct WebSocket from feature components — use this module.
 */
export function connectRealtime(token: string, handlers: RealtimeHandlers): () => void {
  let closed = false;
  let ws: WebSocket | null = null;
  let retry = 0;
  let timer: number | undefined;

  const base = getApiBase().replace(/^http/, 'ws');
  const url = `${base}/v1/ws?token=${encodeURIComponent(token)}`;

  function setStatus(s: RealtimeStatus) {
    handlers.onStatus?.(s);
  }

  function open() {
    if (closed) return;
    setStatus('connecting');
    ws = new WebSocket(url);
    activeWs = ws;
    ws.onopen = () => {
      retry = 0;
      setStatus('connected');
    };
    ws.onmessage = (ev) => {
      try {
        const frame = JSON.parse(String(ev.data)) as { type?: string; payload?: unknown };
        if (!frame.type || !frame.payload) return;
        switch (frame.type) {
          case 'message.created':
            handlers.onMessageCreated?.(frame.payload as Message);
            break;
          case 'message.read':
            handlers.onMessageRead?.(
              frame.payload as { conversation_id: string; reader_id: string; last_read_seq: number },
            );
            break;
          case 'typing.started':
            handlers.onTypingStarted?.(
              frame.payload as { conversation_id: string; user_id: string },
            );
            break;
          case 'typing.stopped':
            handlers.onTypingStopped?.(
              frame.payload as { conversation_id: string; user_id: string },
            );
            break;
        }
      } catch {
        // ignore malformed
      }
    };
    ws.onclose = () => {
      if (activeWs === ws) activeWs = null;
      setStatus('offline');
      if (!closed) {
        const delay = Math.min(10000, 500 * 2 ** retry);
        retry += 1;
        timer = window.setTimeout(open, delay);
      }
    };
    ws.onerror = () => {
      ws?.close();
    };
  }

  open();

  return () => {
    closed = true;
    if (timer) window.clearTimeout(timer);
    ws?.close();
    if (activeWs === ws) activeWs = null;
    setStatus('idle');
  };
}

/**
 * Send a frame to the server over the active WebSocket.
 * Silently no-ops if the socket is not connected.
 */
export function sendFrame(type: string, payload: unknown): void {
  if (!activeWs || activeWs.readyState !== WebSocket.OPEN) return;
  activeWs.send(JSON.stringify({ type, payload }));
}
