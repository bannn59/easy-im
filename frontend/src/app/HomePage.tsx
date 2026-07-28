export function HomePage() {
  return (
    <section>
      <h1>easy-im scaffold</h1>
      <p>
        Monorepo shell only — no chat product features yet. Backend health lives
        at <code>GET /healthz</code>; use the API health page to probe it from
        the browser.
      </p>
      <ul>
        <li>
          Backend: <code>cd backend && go run ./cmd/api</code>
        </li>
        <li>
          Frontend: <code>cd frontend && npm run dev</code>
        </li>
      </ul>
    </section>
  );
}
