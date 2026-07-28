package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"easy-im/backend/internal/apperr"
	"easy-im/backend/internal/domain"
)

type memUsers struct {
	byID  map[string]domain.UserRecord
	email map[string]string
}

func newMemUsers() *memUsers {
	return &memUsers{
		byID:  map[string]domain.UserRecord{},
		email: map[string]string{},
	}
}

func (m *memUsers) Create(_ context.Context, rec domain.UserRecord) error {
	if _, ok := m.email[rec.Email]; ok {
		return apperr.Conflict("email already registered")
	}
	m.byID[rec.ID] = rec
	m.email[rec.Email] = rec.ID
	return nil
}

func (m *memUsers) FindByEmail(_ context.Context, email string) (domain.UserRecord, error) {
	id, ok := m.email[email]
	if !ok {
		return domain.UserRecord{}, apperr.NotFound("user not found")
	}
	return m.byID[id], nil
}

func (m *memUsers) FindByID(_ context.Context, id string) (domain.User, error) {
	rec, ok := m.byID[id]
	if !ok {
		return domain.User{}, apperr.NotFound("user not found")
	}
	return rec.User, nil
}

func testAuth(t *testing.T) *AuthService {
	t.Helper()
	return NewAuthService(newMemUsers(), AuthConfig{
		JWTSecret: []byte("test-secret-at-least-32-bytes-long!!"),
		TokenTTL:  time.Hour,
	})
}

func TestRegisterLoginMe(t *testing.T) {
	svc := testAuth(t)
	ctx := context.Background()

	reg, err := svc.Register(ctx, "User@Example.COM", "password12")
	if err != nil {
		t.Fatal(err)
	}
	if reg.User.Email != "user@example.com" {
		t.Fatalf("email = %q", reg.User.Email)
	}
	if reg.AccessToken == "" {
		t.Fatal("missing token")
	}

	_, err = svc.Register(ctx, "user@example.com", "password12")
	if !errors.Is(err, apperr.ErrConflict) {
		t.Fatalf("want conflict, got %v", err)
	}

	login, err := svc.Login(ctx, "user@example.com", "password12")
	if err != nil {
		t.Fatal(err)
	}
	uid, err := svc.ParseAccessToken(login.AccessToken)
	if err != nil {
		t.Fatal(err)
	}
	me, err := svc.Me(ctx, uid)
	if err != nil {
		t.Fatal(err)
	}
	if me.ID != reg.User.ID {
		t.Fatalf("me id = %s want %s", me.ID, reg.User.ID)
	}
}

func TestLoginBadPassword(t *testing.T) {
	svc := testAuth(t)
	ctx := context.Background()
	if _, err := svc.Register(ctx, "a@b.co", "password12"); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.Login(ctx, "a@b.co", "wrongpass"); !errors.Is(err, apperr.ErrUnauthorized) {
		t.Fatalf("want unauthorized, got %v", err)
	}
	if _, err := svc.Login(ctx, "missing@b.co", "password12"); !errors.Is(err, apperr.ErrUnauthorized) {
		t.Fatalf("want unauthorized for missing, got %v", err)
	}
}

func TestRegisterValidation(t *testing.T) {
	svc := testAuth(t)
	ctx := context.Background()
	if _, err := svc.Register(ctx, "not-an-email", "password12"); !errors.Is(err, apperr.ErrInvalid) {
		t.Fatalf("want invalid email, got %v", err)
	}
	if _, err := svc.Register(ctx, "ok@e.com", "short"); !errors.Is(err, apperr.ErrInvalid) {
		t.Fatalf("want invalid password, got %v", err)
	}
}

func TestAuthNotConfigured(t *testing.T) {
	svc := NewAuthService(newMemUsers(), AuthConfig{})
	if _, err := svc.Register(context.Background(), "a@b.co", "password12"); !errors.Is(err, apperr.ErrUnavailable) {
		t.Fatalf("want unavailable, got %v", err)
	}
}
