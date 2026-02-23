package service

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/ozankenangungor/go-commerce/internal/user/auth"
	"github.com/ozankenangungor/go-commerce/internal/user/repo"
	"github.com/rs/zerolog"
)

// User is a service-layer representation of a user.
type User struct {
	ID        string
	Email     string
	Name      string
	Roles     []string
	CreatedAt time.Time
}

// AuthTokens contains access and refresh token outputs.
type AuthTokens struct {
	AccessToken             string
	RefreshToken            string
	AccessExpiresInSeconds  int64
	RefreshExpiresInSeconds int64
}

// PasswordHasher defines password hashing behavior.
type PasswordHasher interface {
	Hash(password string) (string, error)
	Verify(hash string, password string) error
}

// AccessTokenManager defines JWT access token behavior.
type AccessTokenManager interface {
	SignAccessToken(userID string, roles []string, ttl time.Duration) (string, time.Time, error)
	VerifyAccessToken(accessToken string) (string, []string, error)
}

// Dependencies are required to construct AuthService.
type Dependencies struct {
	Logger                 zerolog.Logger
	UserRepository         repo.UserRepository
	RefreshTokenRepository repo.RefreshTokenRepository
	PasswordHasher         PasswordHasher
	AccessTokenManager     AccessTokenManager
	AccessTokenTTL         time.Duration
	RefreshTokenTTL        time.Duration
}

// AuthService contains authentication business logic.
type AuthService struct {
	logger         zerolog.Logger
	users          repo.UserRepository
	refreshTokens  repo.RefreshTokenRepository
	passwordHasher PasswordHasher
	accessTokens   AccessTokenManager
	accessTokenTTL time.Duration
	refreshTTL     time.Duration
	now            func() time.Time
}

// NewAuthService constructs an AuthService.
func NewAuthService(deps Dependencies) (*AuthService, error) {
	if deps.UserRepository == nil {
		return nil, fmt.Errorf("user repository is required")
	}
	if deps.RefreshTokenRepository == nil {
		return nil, fmt.Errorf("refresh token repository is required")
	}
	if deps.PasswordHasher == nil {
		return nil, fmt.Errorf("password hasher is required")
	}
	if deps.AccessTokenManager == nil {
		return nil, fmt.Errorf("access token manager is required")
	}
	if deps.AccessTokenTTL <= 0 {
		return nil, fmt.Errorf("access token ttl must be > 0")
	}
	if deps.RefreshTokenTTL <= 0 {
		return nil, fmt.Errorf("refresh token ttl must be > 0")
	}

	return &AuthService{
		logger:         deps.Logger,
		users:          deps.UserRepository,
		refreshTokens:  deps.RefreshTokenRepository,
		passwordHasher: deps.PasswordHasher,
		accessTokens:   deps.AccessTokenManager,
		accessTokenTTL: deps.AccessTokenTTL,
		refreshTTL:     deps.RefreshTokenTTL,
		now:            time.Now,
	}, nil
}

// Register creates a user and issues tokens.
func (s *AuthService) Register(ctx context.Context, email, password, name string) (User, AuthTokens, *DomainError, error) {
	normalizedEmail := strings.ToLower(strings.TrimSpace(email))
	trimmedName := strings.TrimSpace(name)
	if normalizedEmail == "" || strings.TrimSpace(password) == "" || trimmedName == "" {
		return User{}, AuthTokens{}, &DomainError{Code: CodeAuthInvalidCredentials, Message: "invalid credentials"}, nil
	}

	passwordHash, err := s.passwordHasher.Hash(password)
	if err != nil {
		return User{}, AuthTokens{}, nil, fmt.Errorf("hash password: %w", err)
	}

	userID, err := auth.NewID(16)
	if err != nil {
		return User{}, AuthTokens{}, nil, fmt.Errorf("generate user id: %w", err)
	}

	createdUser, err := s.users.Create(ctx, repo.CreateUserParams{
		ID:           userID,
		Email:        normalizedEmail,
		Name:         trimmedName,
		PasswordHash: passwordHash,
		Roles:        []string{"customer"},
	})
	if err != nil {
		if errors.Is(err, repo.ErrEmailTaken) {
			return User{}, AuthTokens{}, &DomainError{Code: CodeAuthEmailTaken, Message: "email already taken"}, nil
		}
		return User{}, AuthTokens{}, nil, unavailableError("create user", err)
	}

	tokens, err := s.issueTokens(ctx, createdUser.ID, createdUser.Roles)
	if err != nil {
		return User{}, AuthTokens{}, nil, err
	}

	return mapUser(createdUser), tokens, nil, nil
}

// Login verifies credentials and issues tokens.
func (s *AuthService) Login(ctx context.Context, email, password string) (User, AuthTokens, *DomainError, error) {
	normalizedEmail := strings.ToLower(strings.TrimSpace(email))
	if normalizedEmail == "" || strings.TrimSpace(password) == "" {
		return User{}, AuthTokens{}, &DomainError{Code: CodeAuthInvalidCredentials, Message: "invalid credentials"}, nil
	}

	persistedUser, err := s.users.GetByEmail(ctx, normalizedEmail)
	if err != nil {
		if errors.Is(err, repo.ErrNotFound) {
			return User{}, AuthTokens{}, &DomainError{Code: CodeAuthInvalidCredentials, Message: "invalid credentials"}, nil
		}
		return User{}, AuthTokens{}, nil, unavailableError("get user by email", err)
	}

	if verifyErr := s.passwordHasher.Verify(persistedUser.PasswordHash, password); verifyErr != nil {
		return User{}, AuthTokens{}, &DomainError{Code: CodeAuthInvalidCredentials, Message: "invalid credentials"}, nil
	}

	tokens, err := s.issueTokens(ctx, persistedUser.ID, persistedUser.Roles)
	if err != nil {
		return User{}, AuthTokens{}, nil, err
	}

	return mapUser(persistedUser), tokens, nil, nil
}

// RefreshToken rotates refresh tokens and issues new access/refresh pair.
func (s *AuthService) RefreshToken(ctx context.Context, refreshToken string) (AuthTokens, *DomainError, error) {
	trimmedToken := strings.TrimSpace(refreshToken)
	if trimmedToken == "" {
		return AuthTokens{}, &DomainError{Code: CodeAuthInvalidRefreshToken, Message: "invalid refresh token"}, nil
	}

	tokenHash := auth.HashToken(trimmedToken)
	storedToken, err := s.refreshTokens.GetValidByHash(ctx, tokenHash, s.now().UTC())
	if err != nil {
		if errors.Is(err, repo.ErrNotFound) {
			return AuthTokens{}, &DomainError{Code: CodeAuthInvalidRefreshToken, Message: "invalid refresh token"}, nil
		}
		return AuthTokens{}, nil, unavailableError("get valid refresh token", err)
	}

	persistedUser, err := s.users.GetByID(ctx, storedToken.UserID)
	if err != nil {
		if errors.Is(err, repo.ErrNotFound) {
			return AuthTokens{}, &DomainError{Code: CodeAuthInvalidRefreshToken, Message: "invalid refresh token"}, nil
		}
		return AuthTokens{}, nil, unavailableError("get user by id", err)
	}

	if revokeErr := s.refreshTokens.Revoke(ctx, storedToken.ID, s.now().UTC()); revokeErr != nil {
		if errors.Is(revokeErr, repo.ErrNotFound) {
			return AuthTokens{}, &DomainError{Code: CodeAuthInvalidRefreshToken, Message: "invalid refresh token"}, nil
		}
		return AuthTokens{}, nil, unavailableError("revoke refresh token", revokeErr)
	}

	tokens, err := s.issueTokens(ctx, persistedUser.ID, persistedUser.Roles)
	if err != nil {
		return AuthTokens{}, nil, err
	}

	return tokens, nil, nil
}

// GetProfile returns user profile by id.
func (s *AuthService) GetProfile(ctx context.Context, userID string) (User, *DomainError, error) {
	trimmedUserID := strings.TrimSpace(userID)
	if trimmedUserID == "" {
		return User{}, &DomainError{Code: CodeAuthInvalidCredentials, Message: "invalid credentials"}, nil
	}

	persistedUser, err := s.users.GetByID(ctx, trimmedUserID)
	if err != nil {
		if errors.Is(err, repo.ErrNotFound) {
			return User{}, &DomainError{Code: CodeAuthInvalidCredentials, Message: "invalid credentials"}, nil
		}
		return User{}, nil, unavailableError("get profile", err)
	}

	return mapUser(persistedUser), nil, nil
}

// ValidateAccessToken validates JWT access token and returns identity claims.
func (s *AuthService) ValidateAccessToken(ctx context.Context, accessToken string) (string, []string, *DomainError, error) {
	_ = ctx

	userID, roles, err := s.accessTokens.VerifyAccessToken(accessToken)
	if err != nil {
		if errors.Is(err, auth.ErrExpiredToken) {
			return "", nil, &DomainError{Code: CodeAuthExpiredToken, Message: "token expired"}, nil
		}
		if errors.Is(err, auth.ErrInvalidToken) {
			return "", nil, &DomainError{Code: CodeAuthInvalidToken, Message: "invalid token"}, nil
		}

		s.logger.Error().Err(err).Msg("unexpected token verification error")
		return "", nil, &DomainError{Code: CodeAuthInvalidToken, Message: "invalid token"}, nil
	}

	return userID, roles, nil, nil
}

func (s *AuthService) issueTokens(ctx context.Context, userID string, roles []string) (AuthTokens, error) {
	accessToken, _, err := s.accessTokens.SignAccessToken(userID, roles, s.accessTokenTTL)
	if err != nil {
		return AuthTokens{}, fmt.Errorf("sign access token: %w", err)
	}

	rawRefreshToken, err := auth.NewRefreshToken(32)
	if err != nil {
		return AuthTokens{}, fmt.Errorf("generate refresh token: %w", err)
	}

	refreshTokenID, err := auth.NewID(16)
	if err != nil {
		return AuthTokens{}, fmt.Errorf("generate refresh token id: %w", err)
	}

	refreshTokenExpiry := s.now().UTC().Add(s.refreshTTL)
	if err := s.refreshTokens.Insert(ctx, repo.CreateRefreshTokenParams{
		ID:        refreshTokenID,
		UserID:    userID,
		TokenHash: auth.HashToken(rawRefreshToken),
		ExpiresAt: refreshTokenExpiry,
	}); err != nil {
		return AuthTokens{}, unavailableError("insert refresh token", err)
	}

	return AuthTokens{
		AccessToken:             accessToken,
		RefreshToken:            rawRefreshToken,
		AccessExpiresInSeconds:  int64(s.accessTokenTTL / time.Second),
		RefreshExpiresInSeconds: int64(s.refreshTTL / time.Second),
	}, nil
}

func mapUser(user repo.User) User {
	return User{
		ID:        user.ID,
		Email:     user.Email,
		Name:      user.Name,
		Roles:     append([]string(nil), user.Roles...),
		CreatedAt: user.CreatedAt,
	}
}

func unavailableError(operation string, err error) error {
	return fmt.Errorf("%w: %s: %v", ErrUnavailable, operation, err)
}
