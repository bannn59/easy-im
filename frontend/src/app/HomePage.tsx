import { Link } from 'react-router-dom';
import { useTranslation } from 'react-i18next';
import { useSession } from './Session';

export function HomePage() {
  const session = useSession();
  const { t } = useTranslation();
  return (
    <section className="page">
      <p className="page__eyebrow">{t('home.eyebrow')}</p>
      <h1 className="page__title">{t('home.title')}</h1>
      <p className="page__lead">{t('home.lead')}</p>

      <ul className="page__list">
        <li>
          <strong>{t('home.now')}</strong>
          {t('home.nowBody')}
        </li>
        <li>
          <strong>{t('home.next')}</strong>
          {t('home.nextBody')}
        </li>
      </ul>

      <div className="page__actions">
        {session.user ? (
          <>
            <Link className="btn" to="/app">
              {t('home.openWorkspace')}
            </Link>
            <span className="muted">{t('home.signedInAs', { email: session.user.email })}</span>
          </>
        ) : (
          <>
            <Link className="btn" to="/register">
              {t('home.createAccount')}
            </Link>
            <Link className="btn btn--ghost" to="/login">
              {t('home.signIn')}
            </Link>
          </>
        )}
      </div>
    </section>
  );
}
