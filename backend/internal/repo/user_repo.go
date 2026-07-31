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

// UserRepo persists users in Postgres.
type UserRepo struct {
	pool *pgxpool.Pool
}

func NewUserRepo(pool *pgxpool.Pool) *UserRepo {
	return &UserRepo{pool: pool}
}

func (r *UserRepo) Create(ctx context.Context, rec domain.UserRecord) error {
	_, err := r.pool.Exec(ctx, `
		INSERT INTO users (id, email, display_name, password_hash, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6)
	`, rec.ID, rec.Email, rec.DisplayName, rec.PasswordHash, rec.CreatedAt, rec.UpdatedAt)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return apperr.Conflict("email already registered")
		}
		return apperr.Internal("create user failed", err)
	}
	return nil
}

func (r *UserRepo) FindByEmail(ctx context.Context, email string) (domain.UserRecord, error) {
	var rec domain.UserRecord
	err := r.pool.QueryRow(ctx, `
		SELECT id, email, display_name, password_hash, created_at, updated_at
		FROM users WHERE email = $1
	`, email).Scan(&rec.ID, &rec.Email, &rec.DisplayName, &rec.PasswordHash, &rec.CreatedAt, &rec.UpdatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.UserRecord{}, apperr.NotFound("user not found")
		}
		return domain.UserRecord{}, apperr.Internal("find user by email failed", err)
	}
	return rec, nil
}

func (r *UserRepo) FindByID(ctx context.Context, id string) (domain.User, error) {
	var u domain.User
	err := r.pool.QueryRow(ctx, `
		SELECT id, email, display_name, created_at, updated_at
		FROM users WHERE id = $1
	`, id).Scan(&u.ID, &u.Email, &u.DisplayName, &u.CreatedAt, &u.UpdatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.User{}, apperr.NotFound("user not found")
		}
		return domain.User{}, apperr.Internal("find user by id failed", err)
	}
	return u, nil
}

// FindRecordByID returns the full record (including password hash) for a user.
func (r *UserRepo) FindRecordByID(ctx context.Context, id string) (domain.UserRecord, error) {
	var rec domain.UserRecord
	err := r.pool.QueryRow(ctx, `
		SELECT id, email, display_name, password_hash, created_at, updated_at
		FROM users WHERE id = $1
	`, id).Scan(&rec.ID, &rec.Email, &rec.DisplayName, &rec.PasswordHash, &rec.CreatedAt, &rec.UpdatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.UserRecord{}, apperr.NotFound("user not found")
		}
		return domain.UserRecord{}, apperr.Internal("find user record failed", err)
	}
	return rec, nil
}

// UpdateDisplayName sets the display name and returns the updated user.
func (r *UserRepo) UpdateDisplayName(ctx context.Context, id, displayName string, updatedAt time.Time) (domain.User, error) {
	var u domain.User
	err := r.pool.QueryRow(ctx, `
		UPDATE users
		SET display_name = $2, updated_at = $3
		WHERE id = $1
		RETURNING id, email, display_name, created_at, updated_at
	`, id, displayName, updatedAt).Scan(&u.ID, &u.Email, &u.DisplayName, &u.CreatedAt, &u.UpdatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.User{}, apperr.NotFound("user not found")
		}
		return domain.User{}, apperr.Internal("update display name failed", err)
	}
	return u, nil
}

// UpdatePassword replaces the password hash.
func (r *UserRepo) UpdatePassword(ctx context.Context, id, hash string, updatedAt time.Time) error {
	_, err := r.pool.Exec(ctx, `
		UPDATE users
		SET password_hash = $2, updated_at = $3
		WHERE id = $1
	`, id, hash, updatedAt)
	if err != nil {
		return apperr.Internal("update password failed", err)
	}
	return nil
}

// FindIDsByEmails maps lowercased email → user id for emails that exist.
func (r *UserRepo) FindIDsByEmails(ctx context.Context, emails []string) (map[string]string, error) {
	out := map[string]string{}
	if len(emails) == 0 {
		return out, nil
	}
	rows, err := r.pool.Query(ctx, `
		SELECT id, email FROM users WHERE email = ANY($1)
	`, emails)
	if err != nil {
		return nil, apperr.Internal("lookup emails failed", err)
	}
	defer rows.Close()
	for rows.Next() {
		var id, email string
		if err := rows.Scan(&id, &email); err != nil {
			return nil, apperr.Internal("scan email failed", err)
		}
		out[email] = id
	}
	return out, rows.Err()
}
