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

// FindDirectBetween returns one 1:1 conversation whose members are exactly {userID1, userID2}.
// Preference: latest last_message_at, else latest created_at (R6).
func (r *ConversationRepo) FindDirectBetween(ctx context.Context, userID1, userID2 string) (domain.Conversation, error) {
	var c domain.Conversation
	var title *string
	err := r.pool.QueryRow(ctx, `
		SELECT c.id, c.title, c.created_by, c.created_at, c.updated_at,
			c.last_message_at, c.last_message_seq, c.last_message_preview, c.last_message_sender_id
		FROM conversations c
		WHERE (
			SELECT COUNT(*)::int FROM conversation_members cm WHERE cm.conversation_id = c.id
		) = 2
		AND EXISTS (
			SELECT 1 FROM conversation_members cm
			WHERE cm.conversation_id = c.id AND cm.user_id = $1
		)
		AND EXISTS (
			SELECT 1 FROM conversation_members cm
			WHERE cm.conversation_id = c.id AND cm.user_id = $2
		)
		ORDER BY c.last_message_at DESC NULLS LAST, c.created_at DESC, c.id DESC
		LIMIT 1
	`, userID1, userID2).Scan(
		&c.ID, &title, &c.CreatedBy, &c.CreatedAt, &c.UpdatedAt,
		&c.LastMessageAt, &c.LastMessageSeq, &c.LastMessagePreview, &c.LastMessageSenderID,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.Conversation{}, apperr.NotFound("conversation not found")
		}
		return domain.Conversation{}, apperr.Internal("find direct conversation failed", err)
	}
	c.Title = title
	return c, nil
}

func (r *ConversationRepo) ListForUser(ctx context.Context, userID string) ([]domain.Conversation, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT c.id, c.title, c.created_by, c.created_at, c.updated_at,
			c.last_message_at, c.last_message_seq, c.last_message_preview, c.last_message_sender_id,
			su.email,
			m.last_read_seq,
			(
				SELECT COUNT(*)::bigint FROM messages msg
				WHERE msg.conversation_id = c.id
				  AND msg.seq > m.last_read_seq
				  AND msg.sender_id <> m.user_id
			) AS unread_count,
			(
				SELECT COUNT(*)::int FROM conversation_members cm
				WHERE cm.conversation_id = c.id
			) AS member_count
		FROM conversations c
		INNER JOIN conversation_members m ON m.conversation_id = c.id
		LEFT JOIN users su ON su.id = c.last_message_sender_id
		WHERE m.user_id = $1
		ORDER BY COALESCE(c.last_message_at, c.updated_at) DESC, c.id DESC
	`, userID)
	if err != nil {
		return nil, apperr.Internal("list conversations failed", err)
	}
	defer rows.Close()

	var out []domain.Conversation
	for rows.Next() {
		c, err := scanConversationListRow(rows)
		if err != nil {
			return nil, apperr.Internal("scan conversation failed", err)
		}
		out = append(out, c)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if err := r.attachMembers(ctx, out); err != nil {
		return nil, err
	}
	return out, nil
}

// attachMembers batch-loads members for list rows (sidebar titles need peer emails).
func (r *ConversationRepo) attachMembers(ctx context.Context, list []domain.Conversation) error {
	if len(list) == 0 {
		return nil
	}
	ids := make([]string, len(list))
	index := make(map[string]int, len(list))
	for i, c := range list {
		ids[i] = c.ID
		index[c.ID] = i
	}
	rows, err := r.pool.Query(ctx, `
		SELECT m.conversation_id, u.id, u.email, u.display_name, u.created_at, u.updated_at
		FROM conversation_members m
		INNER JOIN users u ON u.id = m.user_id
		WHERE m.conversation_id = ANY($1)
		ORDER BY m.conversation_id ASC, m.joined_at ASC
	`, ids)
	if err != nil {
		return apperr.Internal("list conversation members failed", err)
	}
	defer rows.Close()
	for rows.Next() {
		var convID string
		var u domain.User
		if err := rows.Scan(&convID, &u.ID, &u.Email, &u.DisplayName, &u.CreatedAt, &u.UpdatedAt); err != nil {
			return apperr.Internal("scan conversation member failed", err)
		}
		i, ok := index[convID]
		if !ok {
			continue
		}
		list[i].Members = append(list[i].Members, u)
	}
	return rows.Err()
}

func scanConversationListRow(row pgx.Row) (domain.Conversation, error) {
	var c domain.Conversation
	var title *string
	err := row.Scan(
		&c.ID, &title, &c.CreatedBy, &c.CreatedAt, &c.UpdatedAt,
		&c.LastMessageAt, &c.LastMessageSeq, &c.LastMessagePreview, &c.LastMessageSenderID,
		&c.LastMessageSenderEmail,
		&c.LastReadSeq, &c.UnreadCount, &c.MemberCount,
	)
	c.Title = title
	return c, err
}

// MarkRead raises the member's last_read_seq to at least seq (clamped by caller).
func (r *ConversationRepo) MarkRead(ctx context.Context, conversationID, userID string, seq int64) (int64, error) {
	var out int64
	err := r.pool.QueryRow(ctx, `
		UPDATE conversation_members
		SET last_read_seq = GREATEST(last_read_seq, $3)
		WHERE conversation_id = $1 AND user_id = $2
		RETURNING last_read_seq
	`, conversationID, userID, seq).Scan(&out)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return 0, apperr.NotFound("conversation not found")
		}
		return 0, apperr.Internal("mark read failed", err)
	}
	return out, nil
}

func (r *ConversationRepo) GetIfMember(ctx context.Context, conversationID, userID string) (domain.Conversation, error) {
	var c domain.Conversation
	var title *string
	err := r.pool.QueryRow(ctx, `
		SELECT c.id, c.title, c.created_by, c.created_at, c.updated_at,
			c.last_message_at, c.last_message_seq, c.last_message_preview, c.last_message_sender_id,
			m.last_read_seq
		FROM conversations c
		INNER JOIN conversation_members m ON m.conversation_id = c.id
		WHERE c.id = $1 AND m.user_id = $2
	`, conversationID, userID).Scan(
		&c.ID, &title, &c.CreatedBy, &c.CreatedAt, &c.UpdatedAt,
		&c.LastMessageAt, &c.LastMessageSeq, &c.LastMessagePreview, &c.LastMessageSenderID,
		&c.LastReadSeq,
	)
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

// ListMemberIDs returns user ids in the conversation.
func (r *ConversationRepo) ListMemberIDs(ctx context.Context, conversationID string) ([]string, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT user_id FROM conversation_members WHERE conversation_id = $1
	`, conversationID)
	if err != nil {
		return nil, apperr.Internal("list member ids failed", err)
	}
	defer rows.Close()
	var out []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, apperr.Internal("scan member id failed", err)
		}
		out = append(out, id)
	}
	return out, rows.Err()
}

func (r *ConversationRepo) listMembers(ctx context.Context, conversationID string) ([]domain.User, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT u.id, u.email, u.display_name, u.created_at, u.updated_at
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
		if err := rows.Scan(&u.ID, &u.Email, &u.DisplayName, &u.CreatedAt, &u.UpdatedAt); err != nil {
			return nil, apperr.Internal("scan member failed", err)
		}
		out = append(out, u)
	}
	return out, rows.Err()
}

