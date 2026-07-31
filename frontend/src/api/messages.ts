import { apiRequest } from './http';

export type ReplyTo = {
  id: string;
  sender_id: string;
  body: string;
};

export type Message = {
  id: string;
  conversation_id: string;
  sender_id: string;
  body: string;
  client_msg_id: string;
  seq: number;
  created_at: string;
  reply_to?: ReplyTo | null;
  edited_at?: string | null;
  recalled_at?: string | null;
};

export function listMessages(
  token: string,
  conversationId: string,
  opts?: { before_seq?: number; limit?: number },
): Promise<{ messages: Message[] }> {
  const q = new URLSearchParams();
  if (opts?.before_seq) q.set('before_seq', String(opts.before_seq));
  if (opts?.limit) q.set('limit', String(opts.limit));
  const qs = q.toString();
  return apiRequest(`/v1/conversations/${conversationId}/messages${qs ? `?${qs}` : ''}`, {
    method: 'GET',
    token,
  });
}

export function sendMessage(
  token: string,
  conversationId: string,
  body: { body: string; client_msg_id: string; reply_to_message_id?: string },
): Promise<Message> {
  return apiRequest(`/v1/conversations/${conversationId}/messages`, {
    method: 'POST',
    token,
    body,
  });
}

export function editMessage(
  token: string,
  conversationId: string,
  messageId: string,
  body: string,
): Promise<Message> {
  return apiRequest(`/v1/conversations/${conversationId}/messages/${messageId}`, {
    method: 'PATCH',
    token,
    body: { body },
  });
}

export function recallMessage(
  token: string,
  conversationId: string,
  messageId: string,
): Promise<Message> {
  return apiRequest(`/v1/conversations/${conversationId}/messages/${messageId}/recall`, {
    method: 'POST',
    token,
    body: {},
  });
}
