package repo

import (
	"context"
	"errors"

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
		INSERT INTO users (id, email, password_hash, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5)
	`, rec.ID, rec.Email, rec.PasswordHash, rec.CreatedAt, rec.UpdatedAt)
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
		SELECT id, email, password_hash, created_at, updated_at
		FROM users WHERE email = $1
	`, email).Scan(&rec.ID, &rec.Email, &rec.PasswordHash, &rec.CreatedAt, &rec.UpdatedAt)
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
		SELECT id, email, created_at, updated_at
		FROM users WHERE id = $1
	`, id).Scan(&u.ID, &u.Email, &u.CreatedAt, &u.UpdatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.User{}, apperr.NotFound("user not found")
		}
		return domain.User{}, apperr.Internal("find user by id failed", err)
	}
	return u, nil
}
