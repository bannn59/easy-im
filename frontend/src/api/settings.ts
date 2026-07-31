import { apiRequest } from './http';
import type { Profile } from './auth';

/** Update the display name. Returns the refreshed profile. */
export function updateProfile(displayName: string): Promise<Profile> {
  return apiRequest<Profile>('/v1/me/profile', {
    method: 'PATCH',
    body: { display_name: displayName },
  });
}

/** Change the password. Requires the current password for verification. */
export function changePassword(
  currentPassword: string,
  newPassword: string,
): Promise<{ ok: boolean }> {
  return apiRequest('/v1/me/password', {
    method: 'POST',
    body: { current_password: currentPassword, new_password: newPassword },
  });
}
