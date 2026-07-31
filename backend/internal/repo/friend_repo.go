package repo

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"easy-im/backend/internal/apperr"
	"easy-im/backend/internal/domain"
)

// FriendRepo persists friend requests and undirected friendships.
type FriendRepo struct {
	pool *pgxpool.Pool
}

func NewFriendRepo(pool *pgxpool.Pool) *FriendRepo {
	return &FriendRepo{pool: pool}
}

// CanonicalPair returns (user_a, user_b) with user_a < user_b by UUID string order.
func CanonicalPair(userID1, userID2 string) (string, string) {
	if userID1 < userID2 {
		return userID1, userID2
	}
	return userID2, userID1
}

func (r *FriendRepo) CreateRequest(ctx context.Context, req domain.FriendRequest) error {
	_, err := r.pool.Exec(ctx, `
		INSERT INTO friend_requests (id, from_user_id, to_user_id, status, created_at, responded_at)
		VALUES ($1, $2, $3, $4, $5, $6)
	`, req.ID, req.FromUserID, req.ToUserID, string(req.Status), req.CreatedAt, req.RespondedAt)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return apperr.Conflict("friend request already pending")
		}
		return apperr.Internal("create friend request failed", err)
	}
	return nil
}

func (r *FriendRepo) GetRequestByID(ctx context.Context, id string) (domain.FriendRequest, error) {
	var req domain.FriendRequest
	var status string
	var respondedAt *time.Time
	err := r.pool.QueryRow(ctx, `
		SELECT id, from_user_id, to_user_id, status, created_at, responded_at
		FROM friend_requests
		WHERE id = $1
	`, id).Scan(&req.ID, &req.FromUserID, &req.ToUserID, &status, &req.CreatedAt, &respondedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.FriendRequest{}, apperr.NotFound("friend request not found")
		}
		return domain.FriendRequest{}, apperr.Internal("get friend request failed", err)
	}
	req.Status = domain.FriendRequestStatus(status)
	req.RespondedAt = respondedAt
	return req, nil
}

// FindPending returns the pending request from fromUserID to toUserID, if any.
func (r *FriendRepo) FindPending(ctx context.Context, fromUserID, toUserID string) (domain.FriendRequest, error) {
	var req domain.FriendRequest
	var status string
	var respondedAt *time.Time
	err := r.pool.QueryRow(ctx, `
		SELECT id, from_user_id, to_user_id, status, created_at, responded_at
		FROM friend_requests
		WHERE from_user_id = $1 AND to_user_id = $2 AND status = 'pending'
	`, fromUserID, toUserID).Scan(&req.ID, &req.FromUserID, &req.ToUserID, &status, &req.CreatedAt, &respondedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.FriendRequest{}, apperr.NotFound("friend request not found")
		}
		return domain.FriendRequest{}, apperr.Internal("find pending friend request failed", err)
	}
	req.Status = domain.FriendRequestStatus(status)
	req.RespondedAt = respondedAt
	return req, nil
}

func (r *FriendRepo) AreFriends(ctx context.Context, userID1, userID2 string) (bool, error) {
	a, b := CanonicalPair(userID1, userID2)
	var n int
	err := r.pool.QueryRow(ctx, `
		SELECT 1 FROM friendships WHERE user_a_id = $1 AND user_b_id = $2
	`, a, b).Scan(&n)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return false, nil
		}
		return false, apperr.Internal("check friendship failed", err)
	}
	return true, nil
}

// ListIncomingPending lists pending requests where userID is the recipient, with from_user hydrated.
func (r *FriendRepo) ListIncomingPending(ctx context.Context, userID string) ([]domain.FriendRequest, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT r.id, r.from_user_id, r.to_user_id, r.status, r.created_at, r.responded_at,
			u.id, u.email, u.created_at, u.updated_at
		FROM friend_requests r
		INNER JOIN users u ON u.id = r.from_user_id
		WHERE r.to_user_id = $1 AND r.status = 'pending'
		ORDER BY r.created_at DESC, r.id DESC
	`, userID)
	if err != nil {
		return nil, apperr.Internal("list incoming friend requests failed", err)
	}
	defer rows.Close()

	var out []domain.FriendRequest
	for rows.Next() {
		var req domain.FriendRequest
		var status string
		var respondedAt *time.Time
		var from domain.User
		if err := rows.Scan(
			&req.ID, &req.FromUserID, &req.ToUserID, &status, &req.CreatedAt, &respondedAt,
			&from.ID, &from.Email, &from.CreatedAt, &from.UpdatedAt,
		); err != nil {
			return nil, apperr.Internal("scan friend request failed", err)
		}
		req.Status = domain.FriendRequestStatus(status)
		req.RespondedAt = respondedAt
		req.FromUser = &from
		out = append(out, req)
	}
	if err := rows.Err(); err != nil {
		return nil, apperr.Internal("iterate friend requests failed", err)
	}
	return out, nil
}

// ListFriends returns peer users for userID, ordered by email.
func (r *FriendRepo) ListFriends(ctx context.Context, userID string) ([]domain.User, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT u.id, u.email, u.created_at, u.updated_at
		FROM friendships f
		INNER JOIN users u ON u.id = CASE
			WHEN f.user_a_id = $1 THEN f.user_b_id
			ELSE f.user_a_id
		END
		WHERE f.user_a_id = $1 OR f.user_b_id = $1
		ORDER BY u.email ASC, u.id ASC
	`, userID)
	if err != nil {
		return nil, apperr.Internal("list friends failed", err)
	}
	defer rows.Close()

	var out []domain.User
	for rows.Next() {
		var u domain.User
		if err := rows.Scan(&u.ID, &u.Email, &u.CreatedAt, &u.UpdatedAt); err != nil {
			return nil, apperr.Internal("scan friend failed", err)
		}
		out = append(out, u)
	}
	if err := rows.Err(); err != nil {
		return nil, apperr.Internal("iterate friends failed", err)
	}
	return out, nil
}

// ListFriendIDs returns peer user IDs for userID, ordered by email.
func (r *FriendRepo) ListFriendIDs(ctx context.Context, userID string) ([]string, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT u.id
		FROM friendships f
		INNER JOIN users u ON u.id = CASE
			WHEN f.user_a_id = $1 THEN f.user_b_id
			ELSE f.user_a_id
		END
		WHERE f.user_a_id = $1 OR f.user_b_id = $1
		ORDER BY u.email ASC, u.id ASC
	`, userID)
	if err != nil {
		return nil, apperr.Internal("list friend ids failed", err)
	}
	defer rows.Close()

	var out []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, apperr.Internal("scan friend id failed", err)
		}
		out = append(out, id)
	}
	if err := rows.Err(); err != nil {
		return nil, apperr.Internal("iterate friend ids failed", err)
	}
	return out, nil
}

// AcceptRequest marks the request accepted and inserts the friendship in one transaction.
func (r *FriendRepo) AcceptRequest(ctx context.Context, requestID, actorUserID string, respondedAt time.Time) (domain.FriendRequest, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return domain.FriendRequest{}, apperr.Internal("begin accept friend tx failed", err)
	}
	defer tx.Rollback(ctx)

	var req domain.FriendRequest
	var status string
	var existingResponded *time.Time
	err = tx.QueryRow(ctx, `
		SELECT id, from_user_id, to_user_id, status, created_at, responded_at
		FROM friend_requests
		WHERE id = $1
		FOR UPDATE
	`, requestID).Scan(&req.ID, &req.FromUserID, &req.ToUserID, &status, &req.CreatedAt, &existingResponded)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.FriendRequest{}, apperr.NotFound("friend request not found")
		}
		return domain.FriendRequest{}, apperr.Internal("lock friend request failed", err)
	}
	req.Status = domain.FriendRequestStatus(status)
	req.RespondedAt = existingResponded

	if req.ToUserID != actorUserID {
		return domain.FriendRequest{}, apperr.Forbidden("not allowed to accept this request")
	}
	if req.Status != domain.FriendRequestPending {
		return domain.FriendRequest{}, apperr.Conflict("friend request is not pending")
	}

	_, err = tx.Exec(ctx, `
		UPDATE friend_requests
		SET status = 'accepted', responded_at = $2
		WHERE id = $1
	`, requestID, respondedAt)
	if err != nil {
		return domain.FriendRequest{}, apperr.Internal("update friend request failed", err)
	}

	a, b := CanonicalPair(req.FromUserID, req.ToUserID)
	_, err = tx.Exec(ctx, `
		INSERT INTO friendships (user_a_id, user_b_id, created_at)
		VALUES ($1, $2, $3)
		ON CONFLICT (user_a_id, user_b_id) DO NOTHING
	`, a, b, respondedAt)
	if err != nil {
		return domain.FriendRequest{}, apperr.Internal("insert friendship failed", err)
	}

	// Clear any reverse pending (race window) so both sides stop seeing dual requests.
	_, err = tx.Exec(ctx, `
		UPDATE friend_requests
		SET status = 'accepted', responded_at = $3
		WHERE from_user_id = $1 AND to_user_id = $2 AND status = 'pending'
	`, req.ToUserID, req.FromUserID, respondedAt)
	if err != nil {
		return domain.FriendRequest{}, apperr.Internal("clear reverse pending failed", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return domain.FriendRequest{}, apperr.Internal("commit accept friend failed", err)
	}

	req.Status = domain.FriendRequestAccepted
	req.RespondedAt = &respondedAt
	return req, nil
}

// RejectRequest marks the request rejected; does not create a friendship.
func (r *FriendRepo) RejectRequest(ctx context.Context, requestID, actorUserID string, respondedAt time.Time) (domain.FriendRequest, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return domain.FriendRequest{}, apperr.Internal("begin reject friend tx failed", err)
	}
	defer tx.Rollback(ctx)

	var req domain.FriendRequest
	var status string
	var existingResponded *time.Time
	err = tx.QueryRow(ctx, `
		SELECT id, from_user_id, to_user_id, status, created_at, responded_at
		FROM friend_requests
		WHERE id = $1
		FOR UPDATE
	`, requestID).Scan(&req.ID, &req.FromUserID, &req.ToUserID, &status, &req.CreatedAt, &existingResponded)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.FriendRequest{}, apperr.NotFound("friend request not found")
		}
		return domain.FriendRequest{}, apperr.Internal("lock friend request failed", err)
	}
	req.Status = domain.FriendRequestStatus(status)
	req.RespondedAt = existingResponded

	if req.ToUserID != actorUserID {
		return domain.FriendRequest{}, apperr.Forbidden("not allowed to reject this request")
	}
	if req.Status != domain.FriendRequestPending {
		return domain.FriendRequest{}, apperr.Conflict("friend request is not pending")
	}

	_, err = tx.Exec(ctx, `
		UPDATE friend_requests
		SET status = 'rejected', responded_at = $2
		WHERE id = $1
	`, requestID, respondedAt)
	if err != nil {
		return domain.FriendRequest{}, apperr.Internal("update friend request failed", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return domain.FriendRequest{}, apperr.Internal("commit reject friend failed", err)
	}

	req.Status = domain.FriendRequestRejected
	req.RespondedAt = &respondedAt
	return req, nil
}
