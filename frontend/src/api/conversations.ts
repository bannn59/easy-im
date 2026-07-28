import { apiRequest } from './http';
import type { PublicUser } from './auth';

export type Conversation = {
  id: string;
  title: string | null;
  created_by: string;
  created_at: string;
  updated_at: string;
  members?: PublicUser[];
};

export function listConversations(token: string): Promise<{ conversations: Conversation[] }> {
  return apiRequest('/v1/conversations', { method: 'GET', token });
}

export function createConversation(
  token: string,
  body: { title?: string; member_emails: string[] },
): Promise<Conversation> {
  return apiRequest('/v1/conversations', { method: 'POST', token, body });
}

export function getConversation(token: string, id: string): Promise<Conversation> {
  return apiRequest(`/v1/conversations/${id}`, { method: 'GET', token });
}
