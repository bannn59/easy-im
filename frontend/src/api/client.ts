import { apiRequest } from './http';

export type HealthzBody = {
  status: string;
};

/** GET /healthz on the API process. */
export async function fetchHealthz(): Promise<HealthzBody> {
  return apiRequest<HealthzBody>('/healthz', { method: 'GET' });
}

export { getApiBase } from './http';
