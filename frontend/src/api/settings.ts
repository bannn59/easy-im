import { apiRequest } from './http';
import type { Profile } from './auth';

/** Update the display name. Returns the refreshed profile. */
export function updateProfile(token: string, displayName: string): Promise<Profile> {
  return apiRequest<Profile>('/v1/me/profile', {
    method: 'PATCH',
    token,
    body: { display_name: displayName },
  });
}

/** Change the password. Requires the current password for verification. */
export function changePassword(
  token: string,
  currentPassword: string,
  newPassword: string,
): Promise<{ ok: boolean }> {
  return apiRequest('/v1/me/password', {
    method: 'POST',
    token,
    body: { current_password: currentPassword, new_password: newPassword },
  });
}
