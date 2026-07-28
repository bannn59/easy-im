package repo

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"easy-im/backend/internal/apperr"
	"easy-im/backend/internal/domain"
)

// ConversationRepo persists conversations and membership.
type ConversationRepo struct {
	pool *pgxpool.Pool
}

func NewConversationRepo(pool *pgxpool.Pool) *ConversationRepo {
	return &ConversationRepo{pool: pool}
}

func (r *ConversationRepo) Create(ctx context.Context, c domain.Conversation, memberIDs []string) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return apperr.Internal("begin tx failed", err)
	}
	defer tx.Rollback(ctx)

	var title any
	if c.Title != nil {
		title = *c.Title
	}
	_, err = tx.Exec(ctx, `
		INSERT INTO conversations (id, title, created_by, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5)
	`, c.ID, title, c.CreatedBy, c.CreatedAt, c.UpdatedAt)
	if err != nil {
		return apperr.Internal("insert conversation failed", err)
	}

	for _, uid := range memberIDs {
		_, err = tx.Exec(ctx, `
			INSERT INTO conversation_members (conversation_id, user_id, joined_at)
			VALUES ($1, $2, $3)
		`, c.ID, uid, c.CreatedAt)
		if err != nil {
			return apperr.Internal("insert member failed", err)
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return apperr.Internal("commit conversation failed", err)
	}
	return nil
}

func (r *ConversationRepo) ListForUser(ctx context.Context, userID string) ([]domain.Conversation, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT c.id, c.title, c.created_by, c.created_at, c.updated_at
		FROM conversations c
		INNER JOIN conversation_members m ON m.conversation_id = c.id
		WHERE m.user_id = $1
		ORDER BY c.updated_at DESC
	`, userID)
	if err != nil {
		return nil, apperr.Internal("list conversations failed", err)
	}
	defer rows.Close()

	var out []domain.Conversation
	for rows.Next() {
		var c domain.Conversation
		var title *string
		if err := rows.Scan(&c.ID, &title, &c.CreatedBy, &c.CreatedAt, &c.UpdatedAt); err != nil {
			return nil, apperr.Internal("scan conversation failed", err)
		}
		c.Title = title
		out = append(out, c)
	}
	return out, rows.Err()
}

func (r *ConversationRepo) GetIfMember(ctx context.Context, conversationID, userID string) (domain.Conversation, error) {
	var c domain.Conversation
	var title *string
	err := r.pool.QueryRow(ctx, `
		SELECT c.id, c.title, c.created_by, c.created_at, c.updated_at
		FROM conversations c
		INNER JOIN conversation_members m ON m.conversation_id = c.id
		WHERE c.id = $1 AND m.user_id = $2
	`, conversationID, userID).Scan(&c.ID, &title, &c.CreatedBy, &c.CreatedAt, &c.UpdatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.Conversation{}, apperr.NotFound("conversation not found")
		}
		return domain.Conversation{}, apperr.Internal("get conversation failed", err)
	}
	c.Title = title

	members, err := r.listMembers(ctx, conversationID)
	if err != nil {
		return domain.Conversation{}, err
	}
	c.Members = members
	return c, nil
}

func (r *ConversationRepo) listMembers(ctx context.Context, conversationID string) ([]domain.User, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT u.id, u.email, u.created_at, u.updated_at
		FROM conversation_members m
		INNER JOIN users u ON u.id = m.user_id
		WHERE m.conversation_id = $1
		ORDER BY m.joined_at ASC
	`, conversationID)
	if err != nil {
		return nil, apperr.Internal("list members failed", err)
	}
	defer rows.Close()
	var out []domain.User
	for rows.Next() {
		var u domain.User
		if err := rows.Scan(&u.ID, &u.Email, &u.CreatedAt, &u.UpdatedAt); err != nil {
			return nil, apperr.Internal("scan member failed", err)
		}
		out = append(out, u)
	}
	return out, rows.Err()
}

