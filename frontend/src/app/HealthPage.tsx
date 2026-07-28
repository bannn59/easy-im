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
    <section className="page">
      <p className="page__eyebrow">System</p>
      <h1 className="page__title">API status</h1>
      <p className="page__lead">
        A live probe of the backend liveness endpoint. No decoration, just the signal.
      </p>

      <div className="panel" aria-live="polite">
        <div className="panel__row">
          <span className="panel__key">Endpoint</span>
          <span className="panel__val">
            <code>
              {base}/healthz
            </code>
          </span>
        </div>
        <div className="panel__row">
          <span className="panel__key">Result</span>
          <span className="panel__val">
            {state.kind === 'loading' && <span className="loading">Checking…</span>}
            {state.kind === 'ok' && (
              <span className="ok">
                {state.status}
              </span>
            )}
            {state.kind === 'error' && <span className="err">{state.message}</span>}
          </span>
        </div>
      </div>

      {state.kind === 'error' && (
        <p className="page__body">
          Start the API with <code>cd backend && go run ./cmd/api</code>, then refresh this page.
        </p>
      )}
    </section>
  );
}
