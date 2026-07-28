import { useEffect, useState } from 'react';
import { useTranslation } from 'react-i18next';
import { fetchHealthz, getApiBase } from '../api/client';

type ProbeState =
  | { kind: 'loading' }
  | { kind: 'ok'; status: string }
  | { kind: 'error'; message: string };

export function HealthPage() {
  const { t } = useTranslation();
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
      <p className="page__eyebrow">{t('health.eyebrow')}</p>
      <h1 className="page__title">{t('health.title')}</h1>
      <p className="page__lead">{t('health.lead')}</p>

      <div className="panel" aria-live="polite">
        <div className="panel__row">
          <span className="panel__key">{t('health.endpoint')}</span>
          <span className="panel__val">
            <code>
              {base}/healthz
            </code>
          </span>
        </div>
        <div className="panel__row">
          <span className="panel__key">{t('health.result')}</span>
          <span className="panel__val">
            {state.kind === 'loading' && <span className="loading">{t('health.checking')}</span>}
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
          {t('health.startApiBefore')} <code>cd backend && go run ./cmd/api</code>
          {t('health.startApiAfter')}
        </p>
      )}
    </section>
  );
}
