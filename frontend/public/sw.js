/* easy-im service worker: renders Web Push notifications and opens the
 * conversation on click. Registered from the settings page. */

/** SW version bump forces an update-check on the browser. */
const SW_VERSION = '1';

self.addEventListener('install', () => {
  self.skipWaiting();
});

self.addEventListener('activate', (event) => {
  event.waitUntil(self.clients.claim());
});

self.addEventListener('push', (event) => {
  let data = {};
  try {
    data = event.data ? event.data.json() : {};
  } catch {
    // non-JSON payload; fall through to defaults
  }
  const payload = data as {
    title?: string;
    body?: string;
    conversation_id?: string;
    tag?: string;
    count?: number;
  };

  const title = payload.title || 'easy-im';
  const options = {
    body: payload.body || 'New message',
    tag: payload.tag || payload.conversation_id || 'easy-im',
    data: { url: `/app/c/${payload.conversation_id ?? ''}`, conversation_id: payload.conversation_id },
    icon: '/icons/icon-192.png',
    badge: '/icons/icon-96.png',
  };
  event.waitUntil(self.registration.showNotification(title, options));
});

self.addEventListener('notificationclick', (event) => {
  event.notification.close();
  const url = (event.notification.data && event.notification.data.url) || '/app';
  event.waitUntil(
    self.clients.matchAll({ type: 'window', includeUncontrolled: true }).then((clients) => {
      for (const client of clients) {
        if ('focus' in client) {
          client.postMessage({ type: 'open-conversation', conversationId: event.notification.data.conversation_id });
          client.focus();
          return;
        }
      }
      if (self.clients.openWindow) {
        return self.clients.openWindow(url);
      }
    }),
  );
});
