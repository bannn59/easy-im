import { apiRequest } from './http';
import type { PublicUser } from './auth';

export type LastMessage = {
  seq: number;
  body: string;
  sender_id: string;
  sender_email?: string | null;
  created_at: string;
};

export type Conversation = {
  id: string;
  title: string | null;
  created_by: string;
  created_at: string;
  updated_at: string;
  members?: PublicUser[];
  last_message?: LastMessage | null;
  unread_count?: number;
  member_count?: number;
};

export function listConversations(): Promise<{ conversations: Conversation[] }> {
  return apiRequest('/v1/conversations', { method: 'GET' });
}

export function getConversation(id: string): Promise<Conversation> {
  return apiRequest(`/v1/conversations/${id}`, { method: 'GET' });
}

export function markConversationRead(
  conversationId: string,
  seq?: number,
): Promise<{ last_read_seq: number; unread_count: number }> {
  const body = seq !== undefined ? { seq } : {};
  return apiRequest(`/v1/conversations/${conversationId}/read`, {
    method: 'POST',
    body,
  });
}
