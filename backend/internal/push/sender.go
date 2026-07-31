package push

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	webpush "github.com/wuc656/webpush-go"
)

// Sender delivers Web Push notifications to browser subscriptions.
type Sender struct {
	// VAPIDPublicKey / VAPIDPrivateKey are the base64url-encoded VAPID keys.
	VAPIDPublicKey  string
	VAPIDPrivateKey string
	// Subject is the VAPID "sub" claim (mailto: or https URL).
	Subject string
	// HTTPClient is optional; defaults to a standard client with a timeout.
	HTTPClient *http.Client
}

// SendResult classifies one delivery attempt.
type SendResult struct {
	// OK is true when the push service accepted the notification.
	OK bool
	// Gone is true when the subscription is no longer valid (410 / 404).
	// Callers should delete the subscription.
	Gone bool
	// Err is non-nil for transient/unexpected failures.
	Err error
}

// NewSender validates required VAPID config and returns a Sender.
func NewSender(vapidPublicKey, vapidPrivateKey, subject string) (*Sender, error) {
	if vapidPublicKey == "" || vapidPrivateKey == "" || subject == "" {
		return nil, errors.New("push: VAPID public key, private key, and subject are required")
	}
	return &Sender{
		VAPIDPublicKey:  vapidPublicKey,
		VAPIDPrivateKey: vapidPrivateKey,
		Subject:         subject,
		HTTPClient:      &http.Client{Timeout: 10 * time.Second},
	}, nil
}

// Send delivers payload to one subscription. Payload is an already-marshalled
// JSON object (the notification data the service worker shows).
func (s *Sender) Send(ctx context.Context, endpoint, p256dh, auth string, payload []byte) SendResult {
	if s == nil || s.HTTPClient == nil {
		return SendResult{OK: false, Err: errors.New("push: sender not configured")}
	}
	sub := &webpush.Subscription{
		Endpoint: endpoint,
		Keys: webpush.Keys{
			Auth:   auth,
			P256dh: p256dh,
		},
	}
	opts := &webpush.Options{
		Subscriber:      s.Subject,
		VAPIDPublicKey:  s.VAPIDPublicKey,
		VAPIDPrivateKey: s.VAPIDPrivateKey,
		TTL:             60, // seconds; push services hold offline notifications this long
		Urgency:         webpush.UrgencyHigh,
	}
	reqCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	resp, err := webpush.SendNotificationWithContext(reqCtx, payload, sub, opts)
	if err != nil {
		return SendResult{OK: false, Err: fmt.Errorf("send push: %w", err)}
	}
	defer resp.Body.Close()

	switch resp.StatusCode {
	case http.StatusCreated, http.StatusOK, http.StatusNoContent:
		return SendResult{OK: true}
	case http.StatusGone, http.StatusNotFound:
		return SendResult{OK: false, Gone: true}
	default:
		return SendResult{OK: false, Err: fmt.Errorf("push service returned %d", resp.StatusCode)}
	}
}
