package auth

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

var (
	// ErrInvalidToken is returned when JWT validation fails for non-expiration reasons.
	ErrInvalidToken = errors.New("invalid token")
	// ErrExpiredToken is returned when JWT validation fails due to expiration.
	ErrExpiredToken = errors.New("expired token")
)

// AccessTokenClaims represents JWT claims for access tokens.
type AccessTokenClaims struct {
	Roles []string `json:"roles"`
	jwt.RegisteredClaims
}

// JWTManager handles JWT creation and verification.
type JWTManager struct {
	secret []byte
	issuer string
	now    func() time.Time
}

// NewJWTManager constructs a JWT manager.
func NewJWTManager(secret string, issuer string) (*JWTManager, error) {
	if strings.TrimSpace(secret) == "" {
		return nil, fmt.Errorf("jwt secret cannot be empty")
	}
	if strings.TrimSpace(issuer) == "" {
		return nil, fmt.Errorf("token issuer cannot be empty")
	}

	return &JWTManager{
		secret: []byte(secret),
		issuer: issuer,
		now:    time.Now,
	}, nil
}

// SignAccessToken signs an HS256 access token.
func (m *JWTManager) SignAccessToken(userID string, roles []string, ttl time.Duration) (string, time.Time, error) {
	if strings.TrimSpace(userID) == "" {
		return "", time.Time{}, fmt.Errorf("user id is required")
	}
	if ttl <= 0 {
		return "", time.Time{}, fmt.Errorf("access token ttl must be > 0")
	}

	now := m.now().UTC()
	expiresAt := now.Add(ttl)
	jti, err := NewID(16)
	if err != nil {
		return "", time.Time{}, fmt.Errorf("generate jti: %w", err)
	}

	claims := AccessTokenClaims{
		Roles: append([]string(nil), roles...),
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   userID,
			Issuer:    m.issuer,
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(expiresAt),
			ID:        jti,
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signedToken, err := token.SignedString(m.secret)
	if err != nil {
		return "", time.Time{}, fmt.Errorf("sign access token: %w", err)
	}

	return signedToken, expiresAt, nil
}

// VerifyAccessToken verifies an HS256 access token and returns identity claims.
func (m *JWTManager) VerifyAccessToken(accessToken string) (string, []string, error) {
	if strings.TrimSpace(accessToken) == "" {
		return "", nil, ErrInvalidToken
	}

	claims := &AccessTokenClaims{}
	parser := jwt.NewParser(
		jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}),
		jwt.WithIssuer(m.issuer),
		jwt.WithTimeFunc(m.now),
	)

	token, err := parser.ParseWithClaims(accessToken, claims, func(token *jwt.Token) (any, error) {
		return m.secret, nil
	})
	if err != nil {
		if errors.Is(err, jwt.ErrTokenExpired) {
			return "", nil, ErrExpiredToken
		}
		return "", nil, ErrInvalidToken
	}

	if !token.Valid || strings.TrimSpace(claims.Subject) == "" {
		return "", nil, ErrInvalidToken
	}

	return claims.Subject, append([]string(nil), claims.Roles...), nil
}

// NewRefreshToken generates an opaque token.
func NewRefreshToken(byteLen int) (string, error) {
	return NewID(byteLen)
}

// HashToken returns SHA-256 hash bytes of an opaque token.
func HashToken(token string) []byte {
	sum := sha256.Sum256([]byte(token))
	hash := make([]byte, len(sum))
	copy(hash, sum[:])
	return hash
}

// NewID generates a random hex identifier with byteLen randomness.
func NewID(byteLen int) (string, error) {
	if byteLen <= 0 {
		return "", fmt.Errorf("byte length must be > 0")
	}

	raw := make([]byte, byteLen)
	if _, err := rand.Read(raw); err != nil {
		return "", fmt.Errorf("generate random bytes: %w", err)
	}

	return hex.EncodeToString(raw), nil
}
