import { BrowserRouter, NavLink, Route, Routes, Link } from 'react-router-dom';
import { AuthPage } from './AuthPage';
import { AppShell, ConversationHome, ConversationRoom } from './AppShell';
import { HealthPage } from './HealthPage';
import { HomePage } from './HomePage';
import { SessionProvider, useSession } from './Session';

function Header() {
  const session = useSession();
  return (
    <header className="shell__header">
      <Link to="/" className="shell__brand">
        easy-im
      </Link>
      <nav className="shell__nav" aria-label="Primary">
        <NavLink to="/" end>
          Home
        </NavLink>
        <NavLink to="/health">Status</NavLink>
        {session.user ? (
          <>
            <NavLink to="/app">Workspace</NavLink>
            <button type="button" className="linkish" onClick={() => session.logout()}>
              Sign out
            </button>
          </>
        ) : (
          <>
            <NavLink to="/login">Sign in</NavLink>
            <NavLink to="/register">Register</NavLink>
          </>
        )}
      </nav>
    </header>
  );
}

export function App() {
  return (
    <SessionProvider>
      <BrowserRouter>
        <div className="shell">
          <Header />
          <main className="shell__main">
            <Routes>
              <Route path="/" element={<HomePage />} />
              <Route path="/health" element={<HealthPage />} />
              <Route path="/login" element={<AuthPage mode="login" />} />
              <Route path="/register" element={<AuthPage mode="register" />} />
              <Route path="/app" element={<AppShell />}>
                <Route index element={<ConversationHome />} />
                <Route path="c/:id" element={<ConversationRoom />} />
              </Route>
            </Routes>
          </main>
        </div>
      </BrowserRouter>
    </SessionProvider>
  );
}
