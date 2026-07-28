import { Link } from 'react-router-dom';
import { useSession } from './Session';

export function HomePage() {
  const session = useSession();
  return (
    <section className="page">
      <p className="page__eyebrow">Instant messaging</p>
      <h1 className="page__title">A quiet place to talk.</h1>
      <p className="page__lead">
        easy-im is building a focused messaging stack. Milestone one is identity: register, sign
        in, and keep a stable session. Conversations and realtime delivery come next.
      </p>

      <ul className="page__list">
        <li>
          <strong>Now</strong>
          Account, session, and API health.
        </li>
        <li>
          <strong>Next</strong>
          Conversations, message history, then live delivery.
        </li>
      </ul>

      <div className="page__actions">
        {session.user ? (
          <>
            <Link className="btn" to="/app">
              Open workspace
            </Link>
            <span className="muted">Signed in as {session.user.email}</span>
          </>
        ) : (
          <>
            <Link className="btn" to="/register">
              Create account
            </Link>
            <Link className="btn btn--ghost" to="/login">
              Sign in
            </Link>
          </>
        )}
      </div>
    </section>
  );
}
