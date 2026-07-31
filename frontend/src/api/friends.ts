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

export function sendFriendRequest(email: string): Promise<FriendRequest> {
  return apiRequest<FriendRequest>('/v1/friends/requests', {
    method: 'POST',
    body: { email },
  });
}

export function listIncomingFriendRequests(): Promise<{ requests: FriendRequest[] }> {
  return apiRequest('/v1/friends/requests/incoming', { method: 'GET' });
}

export function listFriends(): Promise<{ friends: PublicUser[] }> {
  return apiRequest('/v1/friends', { method: 'GET' });
}

export function acceptFriendRequest(requestId: string): Promise<FriendRequest> {
  return apiRequest(`/v1/friends/requests/${requestId}/accept`, {
    method: 'POST',
    body: {},
  });
}

export function rejectFriendRequest(requestId: string): Promise<FriendRequest> {
  return apiRequest(`/v1/friends/requests/${requestId}/reject`, {
    method: 'POST',
    body: {},
  });
}

/** Get-or-create a 1:1 conversation with an accepted friend. */
export function openFriendConversation(peerUserId: string): Promise<Conversation> {
  return apiRequest(`/v1/friends/${peerUserId}/conversation`, {
    method: 'POST',
    body: {},
  });
}
