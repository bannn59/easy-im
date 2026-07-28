import { useEffect, useState } from 'react';
import { fetchHealthz, getApiBase } from '../api/client';

type ProbeState =
  | { kind: 'loading' }
  | { kind: 'ok'; status: string }
  | { kind: 'error'; message: string };

export function HealthPage() {
  const [state, setState] = useState<ProbeState>({ kind: 'loading' });
  const base = getApiBase();

  useEffect(() => {
    let cancelled = false;
    setState({ kind: 'loading' });
    fetchHealthz()
      .then((body) => {
        if (!cancelled) setState({ kind: 'ok', status: body.status });
      })
      .catch((err: unknown) => {
        if (!cancelled) {
          const message = err instanceof Error ? err.message : String(err);
          setState({ kind: 'error', message });
        }
      });
    return () => {
      cancelled = true;
    };
  }, []);

  return (
    <section>
      <h1>API health</h1>
      <p>
        Probing <code>{base}/healthz</code>
      </p>
      {state.kind === 'loading' && <p>Loading…</p>}
      {state.kind === 'ok' && (
        <p className="ok">
          status: <strong>{state.status}</strong>
        </p>
      )}
      {state.kind === 'error' && (
        <p className="err">
          Failed: {state.message}. Start the API with{' '}
          <code>cd backend && go run ./cmd/api</code>.
        </p>
      )}
    </section>
  );
}
