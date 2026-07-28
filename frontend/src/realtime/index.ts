import { getApiBase } from '../api/http';
import type { Message } from '../api/messages';

export type RealtimeStatus = 'idle' | 'connecting' | 'connected' | 'offline';

export type RealtimeHandlers = {
  onMessageCreated?: (m: Message) => void;
  onStatus?: (s: RealtimeStatus) => void;
};

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
    ws.onopen = () => {
      retry = 0;
      setStatus('connected');
    };
    ws.onmessage = (ev) => {
      try {
        const frame = JSON.parse(String(ev.data)) as { type?: string; payload?: Message };
        if (frame.type === 'message.created' && frame.payload) {
          handlers.onMessageCreated?.(frame.payload);
        }
      } catch {
        // ignore malformed
      }
    };
    ws.onclose = () => {
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
    setStatus('idle');
  };
}
