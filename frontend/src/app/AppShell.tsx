import { Navigate } from 'react-router-dom';
import { useSession } from './Session';

export function AppShell() {
  const session = useSession();

  if (session.loading) {
    return (
      <section className="page">
        <p className="loading">Loading session…</p>
      </section>
    );
  }
  if (!session.user) {
    return <Navigate to="/login" replace />;
  }

  return (
    <section className="page">
      <p className="page__eyebrow">Workspace</p>
      <h1 className="page__title">You are signed in.</h1>
      <p className="page__lead">
        Identity is ready. Conversation list and messaging arrive in later milestones. This shell
        is intentionally quiet so those surfaces can land without fighting decoration.
      </p>

      <div className="panel" aria-label="Session details">
        <div className="panel__row">
          <span className="panel__key">Email</span>
          <span className="panel__val">{session.user.email}</span>
        </div>
        <div className="panel__row">
          <span className="panel__key">User id</span>
          <span className="panel__val">
            <code>{session.user.id}</code>
          </span>
        </div>
      </div>

      <div className="page__actions">
        <button type="button" className="btn btn--ghost" onClick={() => session.logout()}>
          Sign out
        </button>
      </div>
    </section>
  );
}
