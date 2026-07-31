package repo

import (
	"context"
	"errors"
	"time"
	"unicode/utf8"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"easy-im/backend/internal/apperr"
	"easy-im/backend/internal/domain"
)

const lastMessagePreviewMaxRunes = 120

func truncatePreview(s string, max int) string {
	if max <= 0 || utf8.RuneCountInString(s) <= max {
		return s
	}
	return string([]rune(s)[:max])
}

// MessageRepo persists messages.
type MessageRepo struct {
	pool *pgxpool.Pool
}

func NewMessageRepo(pool *pgxpool.Pool) *MessageRepo {
	return &MessageRepo{pool: pool}
}

// IsMember reports whether user is in the conversation.
func (r *ConversationRepo) IsMember(ctx context.Context, conversationID, userID string) (bool, error) {
	var n int
	err := r.pool.QueryRow(ctx, `
		SELECT 1 FROM conversation_members
		WHERE conversation_id = $1 AND user_id = $2
	`, conversationID, userID).Scan(&n)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return false, nil
		}
		return false, apperr.Internal("membership check failed", err)
	}
	return true, nil
}

// TouchUpdatedAt bumps conversation updated_at.
func (r *ConversationRepo) TouchUpdatedAt(ctx context.Context, conversationID string) error {
	_, err := r.pool.Exec(ctx, `UPDATE conversations SET updated_at = now() WHERE id = $1`, conversationID)
	if err != nil {
		return apperr.Internal("touch conversation failed", err)
	}
	return nil
}

func scanMessage(row pgx.Row) (domain.Message, error) {
	var m domain.Message
	err := row.Scan(
		&m.ID, &m.ConversationID, &m.SenderID, &m.Body, &m.ClientMsgID, &m.Seq, &m.CreatedAt, &m.ReplyToMessageID,
		&m.EditedAt, &m.RecalledAt,
	)
	return m, err
}

const messageSelectCols = `id, conversation_id, sender_id, body, client_msg_id, seq, created_at, reply_to_message_id, edited_at, recalled_at`

func (r *MessageRepo) Insert(ctx context.Context, m domain.Message) (domain.Message, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return domain.Message{}, apperr.Internal("begin tx failed", err)
	}
	defer tx.Rollback(ctx)

	var seq int64
	err = tx.QueryRow(ctx, `
		UPDATE conversations SET next_seq = next_seq + 1, updated_at = now()
		WHERE id = $1
		RETURNING next_seq - 1
	`, m.ConversationID).Scan(&seq)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.Message{}, apperr.NotFound("conversation not found")
		}
		return domain.Message{}, apperr.Internal("alloc seq failed", err)
	}
	m.Seq = seq

	_, err = tx.Exec(ctx, `
		INSERT INTO messages (id, conversation_id, sender_id, body, client_msg_id, seq, created_at, reply_to_message_id)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8)
	`, m.ID, m.ConversationID, m.SenderID, m.Body, m.ClientMsgID, m.Seq, m.CreatedAt, m.ReplyToMessageID)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			_ = tx.Rollback(ctx)
			return r.FindByClientMsgID(ctx, m.SenderID, m.ClientMsgID)
		}
		return domain.Message{}, apperr.Internal("insert message failed", err)
	}

	preview := truncatePreview(m.Body, lastMessagePreviewMaxRunes)
	_, err = tx.Exec(ctx, `
		UPDATE conversations SET
			last_message_at = $2,
			last_message_seq = $3,
			last_message_preview = $4,
			last_message_sender_id = $5,
			updated_at = now()
		WHERE id = $1
	`, m.ConversationID, m.CreatedAt, m.Seq, preview, m.SenderID)
	if err != nil {
		return domain.Message{}, apperr.Internal("update conversation head failed", err)
	}

	_, err = tx.Exec(ctx, `
		UPDATE conversation_members
		SET last_read_seq = GREATEST(last_read_seq, $3)
		WHERE conversation_id = $1 AND user_id = $2
	`, m.ConversationID, m.SenderID, m.Seq)
	if err != nil {
		return domain.Message{}, apperr.Internal("advance sender read cursor failed", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return domain.Message{}, apperr.Internal("commit message failed", err)
	}
	return m, nil
}

func (r *MessageRepo) FindByClientMsgID(ctx context.Context, senderID, clientMsgID string) (domain.Message, error) {
	m, err := scanMessage(r.pool.QueryRow(ctx, `
		SELECT `+messageSelectCols+`
		FROM messages WHERE sender_id = $1 AND client_msg_id = $2
	`, senderID, clientMsgID))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.Message{}, apperr.NotFound("message not found")
		}
		return domain.Message{}, apperr.Internal("find message failed", err)
	}
	return m, nil
}

func (r *MessageRepo) FindByID(ctx context.Context, id string) (domain.Message, error) {
	m, err := scanMessage(r.pool.QueryRow(ctx, `
		SELECT `+messageSelectCols+`
		FROM messages WHERE id = $1
	`, id))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.Message{}, apperr.NotFound("message not found")
		}
		return domain.Message{}, apperr.Internal("find message by id failed", err)
	}
	return m, nil
}

func (r *MessageRepo) FindByIDs(ctx context.Context, ids []string) (map[string]domain.Message, error) {
	out := make(map[string]domain.Message, len(ids))
	if len(ids) == 0 {
		return out, nil
	}
	rows, err := r.pool.Query(ctx, `
		SELECT `+messageSelectCols+`
		FROM messages WHERE id = ANY($1)
	`, ids)
	if err != nil {
		return nil, apperr.Internal("find messages by ids failed", err)
	}
	defer rows.Close()
	for rows.Next() {
		m, err := scanMessage(rows)
		if err != nil {
			return nil, apperr.Internal("scan message failed", err)
		}
		out[m.ID] = m
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return out, nil
}

// List returns messages with seq < beforeSeq (or latest if beforeSeq==0), newest-last for UI ascending.
func (r *MessageRepo) List(ctx context.Context, conversationID string, beforeSeq int64, limit int) ([]domain.Message, error) {
	if limit <= 0 || limit > 100 {
		limit = 50
	}
	var rows pgx.Rows
	var err error
	if beforeSeq > 0 {
		rows, err = r.pool.Query(ctx, `
			SELECT `+messageSelectCols+`
			FROM messages
			WHERE conversation_id = $1 AND seq < $2
			ORDER BY seq DESC
			LIMIT $3
		`, conversationID, beforeSeq, limit)
	} else {
		rows, err = r.pool.Query(ctx, `
			SELECT `+messageSelectCols+`
			FROM messages
			WHERE conversation_id = $1
			ORDER BY seq DESC
			LIMIT $2
		`, conversationID, limit)
	}
	if err != nil {
		return nil, apperr.Internal("list messages failed", err)
	}
	defer rows.Close()
	var tmp []domain.Message
	for rows.Next() {
		m, err := scanMessage(rows)
		if err != nil {
			return nil, apperr.Internal("scan message failed", err)
		}
		tmp = append(tmp, m)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	// reverse to ascending seq
	for i, j := 0, len(tmp)-1; i < j; i, j = i+1, j-1 {
		tmp[i], tmp[j] = tmp[j], tmp[i]
	}
	return tmp, nil
}

// UpdateBody edits a message body and its edited_at. Also refreshes the
// conversation list preview when the edited message is the conversation head.
func (r *MessageRepo) UpdateBody(ctx context.Context, id, body string, editedAt time.Time) (domain.Message, error) {
	m, err := scanMessage(r.pool.QueryRow(ctx, `
		UPDATE messages SET body = $2, edited_at = $3
		WHERE id = $1
		RETURNING `+messageSelectCols,
		id, body, editedAt))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.Message{}, apperr.NotFound("message not found")
		}
		return domain.Message{}, apperr.Internal("edit message failed", err)
	}
	r.refreshHeadPreview(ctx, m.ConversationID, m.Seq, truncatePreview(m.Body, lastMessagePreviewMaxRunes))
	return m, nil
}

// MarkRecalled sets recalled_at on a message. Also refreshes the conversation
// list preview to a "[recalled]" marker when the message is the conversation head.
func (r *MessageRepo) MarkRecalled(ctx context.Context, id string, recalledAt time.Time) (domain.Message, error) {
	m, err := scanMessage(r.pool.QueryRow(ctx, `
		UPDATE messages SET recalled_at = $2
		WHERE id = $1
		RETURNING `+messageSelectCols,
		id, recalledAt))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.Message{}, apperr.NotFound("message not found")
		}
		return domain.Message{}, apperr.Internal("recall message failed", err)
	}
	r.refreshHeadPreview(ctx, m.ConversationID, m.Seq, recalledPreview)
	return m, nil
}

const recalledPreview = "[recalled]"

// refreshHeadPreview conditionally updates last_message_preview only when the
// given seq is still the conversation head (edit/recall of an older message
// must not clobber the newer head's preview).
func (r *MessageRepo) refreshHeadPreview(ctx context.Context, conversationID string, seq int64, preview string) {
	_, _ = r.pool.Exec(ctx, `
		UPDATE conversations SET last_message_preview = $3
		WHERE id = $1 AND last_message_seq = $2
	`, conversationID, seq, preview)
}
