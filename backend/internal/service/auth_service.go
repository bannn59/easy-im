package service

import (
	"context"
	"errors"
	"net/mail"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"

	"easy-im/backend/internal/apperr"
	"easy-im/backend/internal/domain"
)

// UserStore is the persistence port for auth.
type UserStore interface {
	Create(ctx context.Context, rec domain.UserRecord) error
	FindByEmail(ctx context.Context, email string) (domain.UserRecord, error)
	FindByID(ctx context.Context, id string) (domain.User, error)
	FindRecordByID(ctx context.Context, id string) (domain.UserRecord, error)
	UpdateDisplayName(ctx context.Context, id, displayName string, updatedAt time.Time) (domain.User, error)
	UpdatePassword(ctx context.Context, id, hash string, updatedAt time.Time) error
}

// AuthConfig holds token signing settings.
type AuthConfig struct {
	JWTSecret []byte
	TokenTTL  time.Duration
}

// AuthService implements register/login/me.
type AuthService struct {
	users UserStore
	cfg   AuthConfig
	// now is overridable in tests.
	now func() time.Time
}

func NewAuthService(users UserStore, cfg AuthConfig) *AuthService {
	if cfg.TokenTTL <= 0 {
		cfg.TokenTTL = 168 * time.Hour
	}
	return &AuthService{
		users: users,
		cfg:   cfg,
		now:   time.Now,
	}
}

// TokenResult is returned after register/login.
type TokenResult struct {
	AccessToken string
	User        domain.User
}

func (s *AuthService) ensureReady() error {
	if len(s.cfg.JWTSecret) == 0 {
		return apperr.Unavailable("auth not configured")
	}
	if s.users == nil {
		return apperr.Unavailable("database not configured")
	}
	return nil
}

func normalizeEmail(email string) (string, error) {
	email = strings.TrimSpace(strings.ToLower(email))
	if email == "" {
		return "", apperr.Invalid("email is required")
	}
	if _, err := mail.ParseAddress(email); err != nil {
		return "", apperr.Invalid("email is invalid")
	}
	// ParseAddress may accept "Name <a@b.com>"; require bare address.
	if strings.Contains(email, " ") || strings.Contains(email, "<") {
		return "", apperr.Invalid("email is invalid")
	}
	return email, nil
}

func validatePassword(password string) error {
	if len(password) < 8 {
		return apperr.Invalid("password must be at least 8 characters")
	}
	return nil
}

func (s *AuthService) Register(ctx context.Context, email, password string) (TokenResult, error) {
	if err := s.ensureReady(); err != nil {
		return TokenResult{}, err
	}
	email, err := normalizeEmail(email)
	if err != nil {
		return TokenResult{}, err
	}
	if err := validatePassword(password); err != nil {
		return TokenResult{}, err
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return TokenResult{}, apperr.Internal("hash password failed", err)
	}

	now := s.now().UTC()
	id := uuid.NewString()
	rec := domain.UserRecord{
		User: domain.User{
			ID:        id,
			Email:     email,
			CreatedAt: now,
			UpdatedAt: now,
		},
		PasswordHash: string(hash),
	}
	if err := s.users.Create(ctx, rec); err != nil {
		return TokenResult{}, err
	}

	token, err := s.issueToken(rec.User)
	if err != nil {
		return TokenResult{}, err
	}
	return TokenResult{AccessToken: token, User: rec.User}, nil
}

func (s *AuthService) Login(ctx context.Context, email, password string) (TokenResult, error) {
	if err := s.ensureReady(); err != nil {
		return TokenResult{}, err
	}
	email, err := normalizeEmail(email)
	if err != nil {
		// Treat bad email shape as unauthorized on login to reduce oracle? Prefer invalid for bad format.
		return TokenResult{}, err
	}
	if password == "" {
		return TokenResult{}, apperr.Unauthorized("invalid email or password")
	}

	rec, err := s.users.FindByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, apperr.ErrNotFound) {
			return TokenResult{}, apperr.Unauthorized("invalid email or password")
		}
		return TokenResult{}, err
	}
	if err := bcrypt.CompareHashAndPassword([]byte(rec.PasswordHash), []byte(password)); err != nil {
		return TokenResult{}, apperr.Unauthorized("invalid email or password")
	}

	token, err := s.issueToken(rec.User)
	if err != nil {
		return TokenResult{}, err
	}
	return TokenResult{AccessToken: token, User: rec.User}, nil
}

func (s *AuthService) Me(ctx context.Context, userID string) (domain.User, error) {
	if err := s.ensureReady(); err != nil {
		return domain.User{}, err
	}
	if userID == "" {
		return domain.User{}, apperr.Unauthorized("missing credentials")
	}
	u, err := s.users.FindByID(ctx, userID)
	if err != nil {
		if errors.Is(err, apperr.ErrNotFound) {
			return domain.User{}, apperr.Unauthorized("invalid credentials")
		}
		return domain.User{}, err
	}
	return u, nil
}

const displayNameMaxRunes = 64

// UpdateDisplayName sets the user's display name (empty clears it).
func (s *AuthService) UpdateDisplayName(ctx context.Context, userID, displayName string) (domain.User, error) {
	if err := s.ensureReady(); err != nil {
		return domain.User{}, err
	}
	if userID == "" {
		return domain.User{}, apperr.Unauthorized("missing credentials")
	}
	name := strings.TrimSpace(displayName)
	if len([]rune(name)) > displayNameMaxRunes {
		return domain.User{}, apperr.Invalid("display name too long")
	}
	return s.users.UpdateDisplayName(ctx, userID, name, s.now().UTC())
}

// ChangePassword verifies the current password and replaces it with newPass.
// Existing tokens remain valid (multi-device revocation is a later concern).
func (s *AuthService) ChangePassword(ctx context.Context, userID, current, newPass string) error {
	if err := s.ensureReady(); err != nil {
		return err
	}
	if userID == "" {
		return apperr.Unauthorized("missing credentials")
	}
	if err := validatePassword(newPass); err != nil {
		return err
	}
	rec, err := s.users.FindRecordByID(ctx, userID)
	if err != nil {
		if errors.Is(err, apperr.ErrNotFound) {
			return apperr.Unauthorized("invalid credentials")
		}
		return err
	}
	if err := bcrypt.CompareHashAndPassword([]byte(rec.PasswordHash), []byte(current)); err != nil {
		return apperr.Unauthorized("current password is incorrect")
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(newPass), bcrypt.DefaultCost)
	if err != nil {
		return apperr.Internal("hash password failed", err)
	}
	return s.users.UpdatePassword(ctx, userID, string(hash), s.now().UTC())
}

type accessClaims struct {
	Email string `json:"email"`
	jwt.RegisteredClaims
}

func (s *AuthService) issueToken(u domain.User) (string, error) {
	now := s.now()
	claims := accessClaims{
		Email: u.Email,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   u.ID,
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(s.cfg.TokenTTL)),
		},
	}
	t := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := t.SignedString(s.cfg.JWTSecret)
	if err != nil {
		return "", apperr.Internal("sign token failed", err)
	}
	return signed, nil
}

// ParseAccessToken returns the user id (sub) from a Bearer token string.
func (s *AuthService) ParseAccessToken(token string) (string, error) {
	if err := s.ensureReady(); err != nil {
		return "", err
	}
	token = strings.TrimSpace(token)
	if token == "" {
		return "", apperr.Unauthorized("missing credentials")
	}

	parsed, err := jwt.ParseWithClaims(token, &accessClaims{}, func(t *jwt.Token) (interface{}, error) {
		if t.Method != jwt.SigningMethodHS256 {
			return nil, apperr.Unauthorized("invalid credentials")
		}
		return s.cfg.JWTSecret, nil
	})
	if err != nil || !parsed.Valid {
		return "", apperr.Unauthorized("invalid credentials")
	}
	claims, ok := parsed.Claims.(*accessClaims)
	if !ok || claims.Subject == "" {
		return "", apperr.Unauthorized("invalid credentials")
	}
	return claims.Subject, nil
}
