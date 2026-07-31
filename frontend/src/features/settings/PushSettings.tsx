import { useCallback, useEffect, useState } from 'react';
import { useTranslation } from 'react-i18next';
import { fetchVAPIDPublicKey, registerPushSubscription, unregisterPushSubscription } from '../../api/push';
import { ApiError } from '../../api/http';

/** Read the browser's PushSubscription into the payload the backend stores. */
function subscriptionPayload(sub: PushSubscription) {
  const p256dh = sub.getKey('p256dh');
  const auth = sub.getKey('auth');
  if (!p256dh || !auth) {
    throw new Error('missing subscription keys');
  }
  return {
    endpoint: sub.endpoint,
    p256dh: btoa(String.fromCharCode(...new Uint8Array(p256dh))),
    auth: btoa(String.fromCharCode(...new Uint8Array(auth))),
  };
}

function base64UrlToBase64(s: string): string {
  const padded = s.replace(/-/g, '+').replace(/_/g, '/');
  return padded + '='.repeat((4 - (padded.length % 4)) % 4);
}

export function PushSettings() {
  const { t } = useTranslation();
  const [enabled, setEnabled] = useState(false);
  const [supported] = useState(() => 'serviceWorker' in navigator && 'PushManager' in window);
  const [busy, setBusy] = useState(false);
  const [notice, setNotice] = useState<string | null>(null);

  // Reflect the current subscription state on mount.
  useEffect(() => {
    let cancelled = false;
    (async () => {
      if (!supported) return;
      try {
        const reg = await navigator.serviceWorker.ready;
        const sub = await reg.pushManager.getSubscription();
        if (!cancelled) setEnabled(!!sub);
      } catch {
        // service worker not active yet
      }
    })();
    return () => {
      cancelled = true;
    };
  }, [supported]);

  const ensureServiceWorker = useCallback(async (): Promise<ServiceWorkerRegistration> => {
    const reg = await navigator.serviceWorker.register('/sw.js');
    // wait until an active worker exists (fresh registration activates async)
    if (reg.active) return reg;
    await new Promise<void>((resolve) => {
      const check = () => {
        if (reg.active) resolve();
        else setTimeout(check, 100);
      };
      check();
    });
    return reg;
  }, []);

  const onEnable = useCallback(async () => {
    setBusy(true);
    setNotice(null);
    try {
      if (Notification.permission !== 'granted') {
        const perm = await Notification.requestPermission();
        if (perm !== 'granted') {
          setNotice(t('settings.pushDenied'));
          return;
        }
      }
      const [reg, vapidKey] = await Promise.all([ensureServiceWorker(), fetchVAPIDPublicKey()]);
      if (!vapidKey) {
        setNotice(t('settings.pushFailed'));
        return;
      }
      let sub = await reg.pushManager.getSubscription();
      if (!sub) {
        sub = await reg.pushManager.subscribe({
          userVisibleOnly: true,
          applicationServerKey: base64UrlToBase64(vapidKey),
        });
      }
      await registerPushSubscription(subscriptionPayload(sub));
      setEnabled(true);
      setNotice(t('settings.pushEnabled'));
    } catch (err) {
      setNotice(err instanceof ApiError ? err.message : t('settings.pushFailed'));
    } finally {
      setBusy(false);
    }
  }, [ensureServiceWorker, t]);

  const onDisable = useCallback(async () => {
    setBusy(true);
    setNotice(null);
    try {
      const reg = await navigator.serviceWorker.ready;
      const sub = await reg.pushManager.getSubscription();
      if (sub) {
        await unregisterPushSubscription(sub.endpoint);
        await sub.unsubscribe();
      }
      setEnabled(false);
      setNotice(t('settings.pushDisabled'));
    } catch (err) {
      setNotice(err instanceof ApiError ? err.message : t('settings.pushFailed'));
    } finally {
      setBusy(false);
    }
  }, [t]);

  return (
    <section className="panel settings__section" aria-labelledby="settings-push-heading">
      <h2 id="settings-push-heading" className="settings__heading">
        {t('settings.pushNotifications')}
      </h2>
      <div className="panel__row">
        <span className="panel__key">{t('settings.pushDescription')}</span>
        <span className="panel__val">
          {!supported ? (
            t('settings.pushUnsupported')
          ) : (
            <button
              className="btn"
              type="button"
              disabled={busy}
              onClick={enabled ? onDisable : onEnable}
            >
              {busy ? t('common.loading') : enabled ? t('settings.pushDisable') : t('settings.pushEnable')}
            </button>
          )}
        </span>
      </div>
      {notice && (
        <p className="ok" role="status">
          {notice}
        </p>
      )}
    </section>
  );
}
