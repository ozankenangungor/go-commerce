package repo

import (
	"context"
	"errors"
	"time"
)

var (
	// ErrNotFound is returned when a row does not exist.
	ErrNotFound = errors.New("not found")
	// ErrEmailTaken is returned when user email uniqueness is violated.
	ErrEmailTaken = errors.New("email already taken")
)

// User represents a persisted user row.
type User struct {
	ID           string
	Email        string
	Name         string
	PasswordHash string
	Roles        []string
	CreatedAt    time.Time
}

// CreateUserParams defines input to create a user.
type CreateUserParams struct {
	ID           string
	Email        string
	Name         string
	PasswordHash string
	Roles        []string
}

// UserRepository handles user persistence.
type UserRepository interface {
	Create(ctx context.Context, params CreateUserParams) (User, error)
	GetByEmail(ctx context.Context, email string) (User, error)
	GetByID(ctx context.Context, userID string) (User, error)
}

// RefreshToken represents a persisted refresh token row.
type RefreshToken struct {
	ID        string
	UserID    string
	TokenHash []byte
	ExpiresAt time.Time
	RevokedAt *time.Time
	CreatedAt time.Time
}

// CreateRefreshTokenParams defines input to insert a refresh token.
type CreateRefreshTokenParams struct {
	ID        string
	UserID    string
	TokenHash []byte
	ExpiresAt time.Time
}

// RefreshTokenRepository handles refresh token persistence.
type RefreshTokenRepository interface {
	Insert(ctx context.Context, params CreateRefreshTokenParams) error
	GetValidByHash(ctx context.Context, tokenHash []byte, now time.Time) (RefreshToken, error)
	Revoke(ctx context.Context, tokenID string, revokedAt time.Time) error
}
