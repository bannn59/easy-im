package push

import "encoding/json"

// NotificationPayload is the JSON delivered to the service worker's push
// event. The worker renders a system notification from these fields.
type NotificationPayload struct {
	Type           string `json:"type"`
	Title          string `json:"title"`
	Body           string `json:"body"`
	ConversationID string `json:"conversation_id"`
	Tag            string `json:"tag"`
	Count          int    `json:"count"`
}

// MarshalPayload encodes a notification payload for webpush Send.
func MarshalPayload(p NotificationPayload) ([]byte, error) {
	return json.Marshal(p)
}
