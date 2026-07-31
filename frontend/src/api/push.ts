import { apiRequest } from './http';

/** Web Push subscription payload sent to the backend. */
export type PushSubscriptionPayload = {
  endpoint: string;
  p256dh: string;
  auth: string;
};

/** Fetch the server VAPID public key for pushManager.subscribe. */
export async function fetchVAPIDPublicKey(): Promise<string> {
  const res = await apiRequest<{ public_key: string }>('/v1/push/vapid', { method: 'GET' });
  return res.public_key ?? '';
}

/** Register the current browser subscription with the backend. */
export async function registerPushSubscription(sub: PushSubscriptionPayload): Promise<void> {
  await apiRequest('/v1/push/subscriptions', { method: 'POST', body: sub });
}

/** Remove a subscription from the backend (idempotent). */
export async function unregisterPushSubscription(endpoint: string): Promise<void> {
  await apiRequest('/v1/push/subscriptions', { method: 'DELETE', body: { endpoint } });
}
