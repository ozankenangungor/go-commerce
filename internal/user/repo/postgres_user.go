package repo

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

// PostgresUserRepository is a pgx-backed user repository.
type PostgresUserRepository struct {
	pool *pgxpool.Pool
}

// NewPostgresUserRepository creates a Postgres user repository.
func NewPostgresUserRepository(pool *pgxpool.Pool) *PostgresUserRepository {
	return &PostgresUserRepository{pool: pool}
}

func (r *PostgresUserRepository) Create(ctx context.Context, params CreateUserParams) (User, error) {
	const query = `
		INSERT INTO users (id, email, name, password_hash, roles)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, email, name, password_hash, roles, created_at
	`

	var user User
	err := r.pool.QueryRow(ctx, query,
		params.ID,
		params.Email,
		params.Name,
		params.PasswordHash,
		params.Roles,
	).Scan(
		&user.ID,
		&user.Email,
		&user.Name,
		&user.PasswordHash,
		&user.Roles,
		&user.CreatedAt,
	)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return User{}, ErrEmailTaken
		}
		return User{}, fmt.Errorf("create user: %w", err)
	}

	return user, nil
}

func (r *PostgresUserRepository) GetByEmail(ctx context.Context, email string) (User, error) {
	const query = `
		SELECT id, email, name, password_hash, roles, created_at
		FROM users
		WHERE email = $1
	`

	var user User
	err := r.pool.QueryRow(ctx, query, email).Scan(
		&user.ID,
		&user.Email,
		&user.Name,
		&user.PasswordHash,
		&user.Roles,
		&user.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return User{}, ErrNotFound
		}
		return User{}, fmt.Errorf("get user by email: %w", err)
	}

	return user, nil
}

func (r *PostgresUserRepository) GetByID(ctx context.Context, userID string) (User, error) {
	const query = `
		SELECT id, email, name, password_hash, roles, created_at
		FROM users
		WHERE id = $1
	`

	var user User
	err := r.pool.QueryRow(ctx, query, userID).Scan(
		&user.ID,
		&user.Email,
		&user.Name,
		&user.PasswordHash,
		&user.Roles,
		&user.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return User{}, ErrNotFound
		}
		return User{}, fmt.Errorf("get user by id: %w", err)
	}

	return user, nil
}
