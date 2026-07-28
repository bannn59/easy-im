import { Link } from 'react-router-dom';
import { useSession } from './Session';

export function HomePage() {
  const session = useSession();
  return (
    <section>
      <h1>easy-im</h1>
      <p>
        Scaffold + auth (M1). Chat product features (conversations, messaging) come in later
        roadmap tasks.
      </p>
      <ul>
        <li>
          Backend: <code>cd backend && go run ./cmd/api</code>
        </li>
        <li>
          Needs DB + <code>AUTH_JWT_SECRET</code> (or <code>AUTH_DEV_INSECURE=1</code>) for auth.
        </li>
      </ul>
      {session.user ? (
        <p>
          Signed in as <strong>{session.user.email}</strong> — open the{' '}
          <Link to="/app">app shell</Link>.
        </p>
      ) : (
        <p>
          <Link to="/login">Sign in</Link> or <Link to="/register">create an account</Link>.
        </p>
      )}
    </section>
  );
}
