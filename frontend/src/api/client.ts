export function getApiBase(): string {
  const raw = import.meta.env.VITE_API_BASE;
  if (typeof raw === 'string' && raw.trim() !== '') {
    return raw.replace(/\/$/, '');
  }
  return 'http://localhost:8080';
}

export type HealthzBody = {
  status: string;
};

/** GET /healthz on the API process. */
export async function fetchHealthz(): Promise<HealthzBody> {
  const res = await fetch(`${getApiBase()}/healthz`);
  if (!res.ok) {
    throw new Error(`HTTP ${res.status}`);
  }
  return (await res.json()) as HealthzBody;
}
