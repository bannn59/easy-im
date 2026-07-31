import { useEffect, useState, type FormEvent } from 'react';
import { Navigate } from 'react-router-dom';
import { useTranslation } from 'react-i18next';
import { fetchMe } from '../../api/auth';
import { changePassword, updateProfile } from '../../api/settings';
import { ApiError } from '../../api/http';
import { useSession } from '../../app/Session';
import { PushSettings } from './PushSettings';

export function SettingsPage() {
  const session = useSession();
  const { t } = useTranslation();

  const [email, setEmail] = useState<string | null>(null);
  const [memberSince, setMemberSince] = useState<string | null>(null);
  const [name, setName] = useState('');
  const [savingName, setSavingName] = useState(false);
  const [nameNotice, setNameNotice] = useState<string | null>(null);

  const [currentPassword, setCurrentPassword] = useState('');
  const [newPassword, setNewPassword] = useState('');
  const [confirmPassword, setConfirmPassword] = useState('');
  const [savingPassword, setSavingPassword] = useState(false);
  const [passwordNotice, setPasswordNotice] = useState<string | null>(null);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    if (!session.user) return;
    fetchMe()
      .then((p) => {
        setEmail(p.email);
        setMemberSince(p.created_at);
        setName(p.display_name ?? '');
      })
      .catch(() => undefined);
  }, [session.user]);

  if (session.loading) {
    return (
      <section className="page">
        <p className="loading">{t('workspace.loadingSession')}</p>
      </section>
    );
  }
  if (!session.user) {
    return <Navigate to="/login" replace />;
  }

  function formatMemberSince(iso: string): string {
    const d = new Date(iso);
    if (Number.isNaN(d.getTime())) return iso;
    return new Intl.DateTimeFormat(undefined, {
      year: 'numeric',
      month: 'short',
      day: 'numeric',
    }).format(d);
  }

  async function onSaveName(e: FormEvent) {
    e.preventDefault();
    if (!session.user) return;
    setSavingName(true);
    setError(null);
    setNameNotice(null);
    try {
      const p = await updateProfile(name);
      setName(p.display_name);
      setEmail(p.email);
      setMemberSince(p.created_at);
      void session.refreshUser();
      setNameNotice(t('settings.nameSaved'));
    } catch (err) {
      setError(err instanceof ApiError ? err.message : t('common.requestFailed'));
    } finally {
      setSavingName(false);
    }
  }

  async function onSavePassword(e: FormEvent) {
    e.preventDefault();
    if (!session.user) return;
    if (newPassword !== confirmPassword) {
      setError(t('settings.passwordMismatch'));
      return;
    }
    setSavingPassword(true);
    setError(null);
    setPasswordNotice(null);
    try {
      await changePassword(currentPassword, newPassword);
      setCurrentPassword('');
      setNewPassword('');
      setConfirmPassword('');
      setPasswordNotice(t('settings.passwordSaved'));
    } catch (err) {
      setError(err instanceof ApiError ? err.message : t('common.requestFailed'));
    } finally {
      setSavingPassword(false);
    }
  }

  return (
    <section className="page settings">
      <p className="page__eyebrow">{t('settings.eyebrow')}</p>
      <h1 className="page__title">{t('settings.title')}</h1>

      <section className="panel settings__section" aria-labelledby="settings-profile-heading">
        <h2 id="settings-profile-heading" className="settings__heading">
          {t('settings.profile')}
        </h2>
        <div className="panel__row">
          <span className="panel__key">{t('settings.email')}</span>
          <span className="panel__val">{email ?? '…'}</span>
        </div>
        <div className="panel__row">
          <span className="panel__key">{t('settings.memberSince')}</span>
          <span className="panel__val">{memberSince ? formatMemberSince(memberSince) : '…'}</span>
        </div>

        <form className="settings__form" onSubmit={onSaveName}>
          <div className="field">
            <label className="field__label" htmlFor="settings-name">
              {t('settings.displayName')}
            </label>
            <input
              id="settings-name"
              className="field__input"
              type="text"
              maxLength={64}
              value={name}
              onChange={(ev) => setName(ev.target.value)}
            />
          </div>
          <div className="page__actions">
            <button className="btn" type="submit" disabled={savingName}>
              {savingName ? t('common.loading') : t('settings.save')}
            </button>
          </div>
        </form>
        {nameNotice && (
          <p className="ok" role="status">
            {nameNotice}
          </p>
        )}
      </section>

      <section className="panel settings__section" aria-labelledby="settings-password-heading">
        <h2 id="settings-password-heading" className="settings__heading">
          {t('settings.changePassword')}
        </h2>
        <form className="settings__form" onSubmit={onSavePassword}>
          <div className="field">
            <label className="field__label" htmlFor="settings-current-pw">
              {t('settings.currentPassword')}
            </label>
            <input
              id="settings-current-pw"
              className="field__input"
              type="password"
              autoComplete="current-password"
              value={currentPassword}
              onChange={(ev) => setCurrentPassword(ev.target.value)}
              required
            />
          </div>
          <div className="field">
            <label className="field__label" htmlFor="settings-new-pw">
              {t('settings.newPassword')}
            </label>
            <input
              id="settings-new-pw"
              className="field__input"
              type="password"
              autoComplete="new-password"
              value={newPassword}
              onChange={(ev) => setNewPassword(ev.target.value)}
              required
            />
          </div>
          <div className="field">
            <label className="field__label" htmlFor="settings-confirm-pw">
              {t('settings.confirmPassword')}
            </label>
            <input
              id="settings-confirm-pw"
              className="field__input"
              type="password"
              autoComplete="new-password"
              value={confirmPassword}
              onChange={(ev) => setConfirmPassword(ev.target.value)}
              required
            />
          </div>
          <div className="page__actions">
            <button className="btn" type="submit" disabled={savingPassword}>
              {savingPassword ? t('common.loading') : t('settings.changePassword')}
            </button>
          </div>
        </form>
        {passwordNotice && (
          <p className="ok" role="status">
            {passwordNotice}
          </p>
        )}
      </section>

      <PushSettings />

      {error && (
        <p className="err" role="alert">
          {error}
        </p>
      )}
    </section>
  );
}
