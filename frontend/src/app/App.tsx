import { BrowserRouter, NavLink, Route, Routes, Link } from 'react-router-dom';
import { useTranslation } from 'react-i18next';
import { LanguageSwitcher } from '../i18n/LanguageSwitcher';
import { AuthPage } from './AuthPage';
import { AppShell, ConversationHome, ConversationRoom } from './AppShell';
import { HealthPage } from './HealthPage';
import { HomePage } from './HomePage';
import { SessionProvider, useSession } from './Session';

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
