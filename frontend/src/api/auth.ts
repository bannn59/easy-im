import { apiRequest } from './http';

export type PublicUser = {
  id: string;
  email: string;
  display_name?: string;
  online?: boolean;
};

export type Profile = {
  id: string;
  email: string;
  display_name: string;
  created_at: string;
};

export type TokenResponse = {
  access_token: string;
  token_type: string;
  user: PublicUser;
};

export function register(email: string, password: string): Promise<TokenResponse> {
  return apiRequest<TokenResponse>('/v1/auth/register', {
    method: 'POST',
    body: { email, password },
  });
}

export function login(email: string, password: string): Promise<TokenResponse> {
  return apiRequest<TokenResponse>('/v1/auth/login', {
    method: 'POST',
    body: { email, password },
  });
}

export function fetchMe(token: string): Promise<Profile> {
  return apiRequest<Profile>('/v1/me', { method: 'GET', token });
}
