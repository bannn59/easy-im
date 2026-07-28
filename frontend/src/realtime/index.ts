/**
 * Realtime (WebSocket) client will live here.
 * Do not open raw WebSocket connections from feature components —
 * see .trellis/spec/frontend/quality-guidelines.md.
 */

export type RealtimeStatus = 'idle' | 'connecting' | 'connected' | 'offline';

/** Placeholder until gateway lands. */
export function getRealtimeStatus(): RealtimeStatus {
  return 'idle';
}
