package repo

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// PostgresRefreshTokenRepository is a pgx-backed refresh token repository.
type PostgresRefreshTokenRepository struct {
	pool *pgxpool.Pool
}

// NewPostgresRefreshTokenRepository creates a Postgres refresh token repository.
func NewPostgresRefreshTokenRepository(pool *pgxpool.Pool) *PostgresRefreshTokenRepository {
	return &PostgresRefreshTokenRepository{pool: pool}
}

func (r *PostgresRefreshTokenRepository) Insert(ctx context.Context, params CreateRefreshTokenParams) error {
	const query = `
		INSERT INTO refresh_tokens (id, user_id, token_hash, expires_at)
		VALUES ($1, $2, $3, $4)
	`

	if _, err := r.pool.Exec(ctx, query,
		params.ID,
		params.UserID,
		params.TokenHash,
		params.ExpiresAt,
	); err != nil {
		return fmt.Errorf("insert refresh token: %w", err)
	}

	return nil
}

func (r *PostgresRefreshTokenRepository) GetValidByHash(ctx context.Context, tokenHash []byte, now time.Time) (RefreshToken, error) {
	const query = `
		SELECT id, user_id, token_hash, expires_at, revoked_at, created_at
		FROM refresh_tokens
		WHERE token_hash = $1
		  AND revoked_at IS NULL
		  AND expires_at > $2
		LIMIT 1
	`

	var token RefreshToken
	err := r.pool.QueryRow(ctx, query, tokenHash, now).Scan(
		&token.ID,
		&token.UserID,
		&token.TokenHash,
		&token.ExpiresAt,
		&token.RevokedAt,
		&token.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return RefreshToken{}, ErrNotFound
		}
		return RefreshToken{}, fmt.Errorf("get valid refresh token by hash: %w", err)
	}

	return token, nil
}

func (r *PostgresRefreshTokenRepository) Revoke(ctx context.Context, tokenID string, revokedAt time.Time) error {
	const query = `
		UPDATE refresh_tokens
		SET revoked_at = $2
		WHERE id = $1 AND revoked_at IS NULL
	`

	result, err := r.pool.Exec(ctx, query, tokenID, revokedAt)
	if err != nil {
		return fmt.Errorf("revoke refresh token: %w", err)
	}
	if result.RowsAffected() == 0 {
		return ErrNotFound
	}

	return nil
}
