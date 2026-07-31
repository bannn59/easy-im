package repo

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"easy-im/backend/internal/apperr"
	"easy-im/backend/internal/domain"
)

// PushSubscriptionRepo persists Web Push subscriptions.
type PushSubscriptionRepo struct {
	pool *pgxpool.Pool
}

func NewPushSubscriptionRepo(pool *pgxpool.Pool) *PushSubscriptionRepo {
	return &PushSubscriptionRepo{pool: pool}
}

// Upsert inserts a subscription, or updates keys when the same (user, endpoint)
// already exists. Returns the stored row.
func (r *PushSubscriptionRepo) Upsert(ctx context.Context, userID, endpoint, p256dh, auth string) (domain.PushSubscription, error) {
	id := uuid.NewString()
	var createdAt, updatedAt [1]time.Time
	err := r.pool.QueryRow(ctx, `
		INSERT INTO push_subscriptions (id, user_id, endpoint, p256dh, auth)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (user_id, endpoint)
		DO UPDATE SET p256dh = EXCLUDED.p256dh, auth = EXCLUDED.auth, updated_at = now()
		RETURNING id, user_id, endpoint, p256dh, auth, created_at, updated_at
	`, id, userID, endpoint, p256dh, auth).Scan(
		&id,
		&userID,
		&endpoint,
		&p256dh,
		&auth,
		&createdAt[0],
		&updatedAt[0],
	)
	if err != nil {
		return domain.PushSubscription{}, apperr.Internal("upsert push subscription failed", err)
	}
	return domain.PushSubscription{
		ID:       id,
		UserID:   userID,
		Endpoint: endpoint,
		P256DH:   p256dh,
		Auth:     auth,
	}, nil
}

// ListByUser returns all push subscriptions for a user.
func (r *PushSubscriptionRepo) ListByUser(ctx context.Context, userID string) ([]domain.PushSubscription, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, user_id, endpoint, p256dh, auth, created_at, updated_at
		FROM push_subscriptions
		WHERE user_id = $1
	`, userID)
	if err != nil {
		return nil, apperr.Internal("list push subscriptions failed", err)
	}
	defer rows.Close()

	var out []domain.PushSubscription
	for rows.Next() {
		var s domain.PushSubscription
		if err := rows.Scan(&s.ID, &s.UserID, &s.Endpoint, &s.P256DH, &s.Auth, &s.CreatedAt, &s.UpdatedAt); err != nil {
			return nil, apperr.Internal("scan push subscription failed", err)
		}
		out = append(out, s)
	}
	if err := rows.Err(); err != nil {
		return nil, apperr.Internal("iterate push subscriptions failed", err)
	}
	return out, nil
}

// DeleteByEndpoint removes one subscription owned by the user (idempotent).
func (r *PushSubscriptionRepo) DeleteByEndpoint(ctx context.Context, userID, endpoint string) error {
	_, err := r.pool.Exec(ctx, `
		DELETE FROM push_subscriptions
		WHERE user_id = $1 AND endpoint = $2
	`, userID, endpoint)
	if err != nil {
		return apperr.Internal("delete push subscription failed", err)
	}
	return nil
}

// DeleteByEndpoints removes multiple subscriptions in one statement, used when
// a push provider reports the endpoint as gone (410 / 404).
func (r *PushSubscriptionRepo) DeleteByEndpoints(ctx context.Context, endpoints []string) error {
	if len(endpoints) == 0 {
		return nil
	}
	_, err := r.pool.Exec(ctx, `
		DELETE FROM push_subscriptions
		WHERE endpoint = ANY($1)
	`, endpoints)
	if err != nil {
		return apperr.Internal("delete push subscriptions failed", err)
	}
	return nil
}
