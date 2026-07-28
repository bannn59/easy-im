import { apiRequest } from './http';
import type { PublicUser } from './auth';
import type { Conversation } from './conversations';

export type FriendRequestStatus = 'pending' | 'accepted' | 'rejected';

export type FriendRequest = {
  id: string;
  from_user_id: string;
  to_user_id: string;
  status: FriendRequestStatus;
  created_at: string;
  responded_at: string | null;
  from_user?: PublicUser;
  to_user?: PublicUser;
};

export function sendFriendRequest(token: string, email: string): Promise<FriendRequest> {
  return apiRequest<FriendRequest>('/v1/friends/requests', {
    method: 'POST',
    token,
    body: { email },
  });
}

export function listIncomingFriendRequests(
  token: string,
): Promise<{ requests: FriendRequest[] }> {
  return apiRequest('/v1/friends/requests/incoming', { method: 'GET', token });
}

export function listFriends(token: string): Promise<{ friends: PublicUser[] }> {
  return apiRequest('/v1/friends', { method: 'GET', token });
}

export function acceptFriendRequest(token: string, requestId: string): Promise<FriendRequest> {
  return apiRequest(`/v1/friends/requests/${requestId}/accept`, {
    method: 'POST',
    token,
    body: {},
  });
}

export function rejectFriendRequest(token: string, requestId: string): Promise<FriendRequest> {
  return apiRequest(`/v1/friends/requests/${requestId}/reject`, {
    method: 'POST',
    token,
    body: {},
  });
}

/** Get-or-create a 1:1 conversation with an accepted friend. */
export function openFriendConversation(token: string, peerUserId: string): Promise<Conversation> {
  return apiRequest(`/v1/friends/${peerUserId}/conversation`, {
    method: 'POST',
    token,
    body: {},
  });
}
