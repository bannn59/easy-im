import { Navigate } from 'react-router-dom';
import { useSession } from './Session';

export function AppShell() {
  const session = useSession();

  if (session.loading) {
    return (
      <section>
        <p>Loading session…</p>
      </section>
    );
  }
  if (!session.user) {
    return <Navigate to="/login" replace />;
  }

  return (
    <section>
      <h1>Signed in</h1>
      <p>
        You are logged in as <strong>{session.user.email}</strong>
      </p>
      <p className="muted">
        User id: <code>{session.user.id}</code>
      </p>
      <p>Conversation list will land in a later task (T3). This page is the M1 shell.</p>
      <button type="button" onClick={() => session.logout()}>
        Sign out
      </button>
    </section>
  );
}
