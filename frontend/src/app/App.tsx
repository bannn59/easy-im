import { BrowserRouter, NavLink, Route, Routes, Link, Navigate } from 'react-router-dom';
import type { ReactNode } from 'react';
import { useTranslation } from 'react-i18next';
import { LanguageSwitcher } from '../i18n/LanguageSwitcher';
import { AuthPage } from './AuthPage';
import { ConversationRoom } from '../features/chat';
import GlobalSearchPanel from '../features/chat/GlobalSearchPanel';
import { FriendsPage } from '../features/friends';
import { SettingsPage } from '../features/settings';
import { AppShell, ConversationHome } from './AppShell';
import { HealthPage } from './HealthPage';
import { HomePage } from './HomePage';
import { SessionProvider, useSession } from './Session';
import { RealtimeProvider } from '../realtime';

function Header() {
  const session = useSession();
  const { t } = useTranslation();
  return (
    <header className="shell__header">
      <Link to="/" className="shell__brand">
        easy-im
      </Link>
      <div className="shell__header-end">
        <nav className="shell__nav" aria-label={t('nav.primary')}>
          <NavLink to="/" end>
            {t('nav.home')}
          </NavLink>
          <NavLink to="/health">{t('nav.status')}</NavLink>
          {session.user ? (
            <>
              <NavLink to="/app">{t('nav.workspace')}</NavLink>
              <NavLink to="/friends">{t('nav.friends')}</NavLink>
              <NavLink to="/search">{t('nav.search')}</NavLink>
              <button type="button" className="linkish" onClick={() => session.logout()}>
                {t('nav.signOut')}
              </button>
            </>
          ) : (
            <>
              <NavLink to="/login">{t('nav.signIn')}</NavLink>
              <NavLink to="/register">{t('nav.register')}</NavLink>
            </>
          )}
        </nav>
        <LanguageSwitcher />
      </div>
    </header>
  );
}

export function App() {
  return (
    <SessionProvider>
      <RealtimeProvider>
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
                <Route path="/friends" element={<FriendsPage />} />
                <Route path="/settings" element={<SettingsPage />} />
                <Route path="/search" element={<RequireAuth><GlobalSearchPanel /></RequireAuth>} />
              </Routes>
            </main>
          </div>
        </BrowserRouter>
      </RealtimeProvider>
    </SessionProvider>
  );
}

/** Guard for routes that need a signed-in user. Redirects to /login. */
function RequireAuth({ children }: { children: ReactNode }) {
  const session = useSession();
  const { t } = useTranslation();
  if (session.loading) {
    return <p className="loading">{t('workspace.loadingSession')}</p>;
  }
  if (!session.user) {
    return <Navigate to="/login" replace />;
  }
  return <>{children}</>;
}
