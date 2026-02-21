package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/ozankenangungor/go-commerce/internal/user/auth"
	"github.com/ozankenangungor/go-commerce/internal/user/repo"
	"github.com/rs/zerolog"
)

type fakeUserRepo struct {
	createFunc     func(ctx context.Context, params repo.CreateUserParams) (repo.User, error)
	getByEmailFunc func(ctx context.Context, email string) (repo.User, error)
	getByIDFunc    func(ctx context.Context, userID string) (repo.User, error)
}

func (f fakeUserRepo) Create(ctx context.Context, params repo.CreateUserParams) (repo.User, error) {
	if f.createFunc == nil {
		return repo.User{}, errors.New("create not configured")
	}
	return f.createFunc(ctx, params)
}

func (f fakeUserRepo) GetByEmail(ctx context.Context, email string) (repo.User, error) {
	if f.getByEmailFunc == nil {
		return repo.User{}, errors.New("get by email not configured")
	}
	return f.getByEmailFunc(ctx, email)
}

func (f fakeUserRepo) GetByID(ctx context.Context, userID string) (repo.User, error) {
	if f.getByIDFunc == nil {
		return repo.User{}, errors.New("get by id not configured")
	}
	return f.getByIDFunc(ctx, userID)
}

type fakeRefreshTokenRepo struct {
	insertFunc         func(ctx context.Context, params repo.CreateRefreshTokenParams) error
	getValidByHashFunc func(ctx context.Context, tokenHash []byte, now time.Time) (repo.RefreshToken, error)
	revokeFunc         func(ctx context.Context, tokenID string, revokedAt time.Time) error
}

func (f fakeRefreshTokenRepo) Insert(ctx context.Context, params repo.CreateRefreshTokenParams) error {
	if f.insertFunc == nil {
		return nil
	}
	return f.insertFunc(ctx, params)
}

func (f fakeRefreshTokenRepo) GetValidByHash(ctx context.Context, tokenHash []byte, now time.Time) (repo.RefreshToken, error) {
	if f.getValidByHashFunc == nil {
		return repo.RefreshToken{}, errors.New("get valid by hash not configured")
	}
	return f.getValidByHashFunc(ctx, tokenHash, now)
}

func (f fakeRefreshTokenRepo) Revoke(ctx context.Context, tokenID string, revokedAt time.Time) error {
	if f.revokeFunc == nil {
		return nil
	}
	return f.revokeFunc(ctx, tokenID, revokedAt)
}

type fakePasswordHasher struct{}

func (f fakePasswordHasher) Hash(password string) (string, error) {
	return "hash", nil
}

func (f fakePasswordHasher) Verify(hash string, password string) error {
	return nil
}

type fakeAccessTokenManager struct {
	signFunc   func(userID string, roles []string, ttl time.Duration) (string, time.Time, error)
	verifyFunc func(accessToken string) (string, []string, error)
}

func (f fakeAccessTokenManager) SignAccessToken(userID string, roles []string, ttl time.Duration) (string, time.Time, error) {
	if f.signFunc == nil {
		return "token", time.Now().Add(ttl), nil
	}
	return f.signFunc(userID, roles, ttl)
}

func (f fakeAccessTokenManager) VerifyAccessToken(accessToken string) (string, []string, error) {
	if f.verifyFunc == nil {
		return "", nil, errors.New("verify not configured")
	}
	return f.verifyFunc(accessToken)
}

func TestAuthServiceValidateAccessTokenDomainMapping(t *testing.T) {
	tests := []struct {
		name      string
		verifyErr error
		wantCode  string
	}{
		{name: "invalid token", verifyErr: auth.ErrInvalidToken, wantCode: CodeAuthInvalidToken},
		{name: "expired token", verifyErr: auth.ErrExpiredToken, wantCode: CodeAuthExpiredToken},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := newTestService(t, fakeAccessTokenManager{
				verifyFunc: func(accessToken string) (string, []string, error) {
					return "", nil, tt.verifyErr
				},
			})

			_, _, domainErr, err := svc.ValidateAccessToken(context.Background(), "token")
			if err != nil {
				t.Fatalf("expected nil infra error, got %v", err)
			}
			if domainErr == nil {
				t.Fatal("expected domain error")
			}
			if domainErr.Code != tt.wantCode {
				t.Fatalf("expected code %s, got %s", tt.wantCode, domainErr.Code)
			}
		})
	}
}

func TestAuthServiceRefreshTokenInvalidMapping(t *testing.T) {
	svc := newTestService(t, fakeAccessTokenManager{
		verifyFunc: func(accessToken string) (string, []string, error) {
			return "", nil, nil
		},
	})
	svc.refreshTokens = fakeRefreshTokenRepo{
		getValidByHashFunc: func(ctx context.Context, tokenHash []byte, now time.Time) (repo.RefreshToken, error) {
			return repo.RefreshToken{}, repo.ErrNotFound
		},
	}

	_, domainErr, err := svc.RefreshToken(context.Background(), "invalid-refresh-token")
	if err != nil {
		t.Fatalf("expected nil infra error, got %v", err)
	}
	if domainErr == nil {
		t.Fatal("expected domain error")
	}
	if domainErr.Code != CodeAuthInvalidRefreshToken {
		t.Fatalf("expected code %s, got %s", CodeAuthInvalidRefreshToken, domainErr.Code)
	}
}

func newTestService(t *testing.T, tokenManager fakeAccessTokenManager) *AuthService {
	t.Helper()

	svc, err := NewAuthService(Dependencies{
		Logger:         zerolog.Nop(),
		UserRepository: fakeUserRepo{getByIDFunc: func(ctx context.Context, userID string) (repo.User, error) { return repo.User{}, repo.ErrNotFound }, getByEmailFunc: func(ctx context.Context, email string) (repo.User, error) { return repo.User{}, repo.ErrNotFound }, createFunc: func(ctx context.Context, params repo.CreateUserParams) (repo.User, error) { return repo.User{}, nil }},
		RefreshTokenRepository: fakeRefreshTokenRepo{getValidByHashFunc: func(ctx context.Context, tokenHash []byte, now time.Time) (repo.RefreshToken, error) {
			return repo.RefreshToken{}, repo.ErrNotFound
		}},
		PasswordHasher:     fakePasswordHasher{},
		AccessTokenManager: tokenManager,
		AccessTokenTTL:     15 * time.Minute,
		RefreshTokenTTL:    720 * time.Hour,
	})
	if err != nil {
		t.Fatalf("new auth service: %v", err)
	}

	return svc
}
