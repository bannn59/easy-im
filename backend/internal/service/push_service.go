package service

import (
	"context"

	"easy-im/backend/internal/apperr"
	"easy-im/backend/internal/domain"
)

// PushSubscriptionStore persists Web Push subscriptions.
type PushSubscriptionStore interface {
	Upsert(ctx context.Context, userID, endpoint, p256dh, auth string) (domain.PushSubscription, error)
	ListByUser(ctx context.Context, userID string) ([]domain.PushSubscription, error)
	DeleteByEndpoint(ctx context.Context, userID, endpoint string) error
}

// PushService manages a user's Web Push subscriptions.
type PushService struct {
	subs PushSubscriptionStore
}

func NewPushService(subs PushSubscriptionStore) *PushService {
	return &PushService{subs: subs}
}

// Register upserts a subscription for the user.
func (s *PushService) Register(ctx context.Context, userID, endpoint, p256dh, auth string) error {
	if userID == "" {
		return apperr.Unauthorized("missing credentials")
	}
	if s.subs == nil {
		return apperr.Unavailable("database not configured")
	}
	if endpoint == "" || p256dh == "" || auth == "" {
		return apperr.Invalid("endpoint, p256dh, and auth are required")
	}
	_, err := s.subs.Upsert(ctx, userID, endpoint, p256dh, auth)
	return err
}

// Unregister removes a subscription for the user (idempotent).
func (s *PushService) Unregister(ctx context.Context, userID, endpoint string) error {
	if userID == "" {
		return apperr.Unauthorized("missing credentials")
	}
	if s.subs == nil {
		return apperr.Unavailable("database not configured")
	}
	if endpoint == "" {
		return apperr.Invalid("endpoint is required")
	}
	return s.subs.DeleteByEndpoint(ctx, userID, endpoint)
}
