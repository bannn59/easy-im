export function getApiBase(): string {
  const raw = import.meta.env.VITE_API_BASE;
  if (typeof raw === 'string' && raw.trim() !== '') {
    return raw.replace(/\/$/, '');
  }
  return 'http://localhost:8080';
}

export type ApiErrorBody = {
  error: {
    code: string;
    message: string;
    request_id?: string;
  };
};

export class ApiError extends Error {
  readonly status: number;
  readonly code: string;
  readonly requestId?: string;

  constructor(status: number, code: string, message: string, requestId?: string) {
    super(message);
    this.name = 'ApiError';
    this.status = status;
    this.code = code;
    this.requestId = requestId;
  }
}

export type RequestOptions = {
  method?: string;
  body?: unknown;
  token?: string | null;
};

export async function apiRequest<T>(path: string, opts: RequestOptions = {}): Promise<T> {
  const headers: Record<string, string> = {
    Accept: 'application/json',
  };
  if (opts.body !== undefined) {
    headers['Content-Type'] = 'application/json';
  }
  if (opts.token) {
    headers.Authorization = `Bearer ${opts.token}`;
  }

  const res = await fetch(`${getApiBase()}${path}`, {
    method: opts.method ?? (opts.body !== undefined ? 'POST' : 'GET'),
    headers,
    body: opts.body !== undefined ? JSON.stringify(opts.body) : undefined,
  });

  if (res.ok) {
    if (res.status === 204) {
      return undefined as T;
    }
    return (await res.json()) as T;
  }

  let code = 'http_error';
  let message = `HTTP ${res.status}`;
  let requestId: string | undefined;
  try {
    const errBody = (await res.json()) as ApiErrorBody;
    if (errBody?.error) {
      code = errBody.error.code || code;
      message = errBody.error.message || message;
      requestId = errBody.error.request_id;
    }
  } catch {
    // non-JSON error
  }
  throw new ApiError(res.status, code, message, requestId);
}
