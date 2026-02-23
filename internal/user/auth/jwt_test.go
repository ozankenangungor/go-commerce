package auth

import (
	"errors"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

func TestJWTManagerSignAndVerify(t *testing.T) {
	manager, err := NewJWTManager("test-secret", "go-commerce")
	if err != nil {
		t.Fatalf("new jwt manager: %v", err)
	}

	token, _, err := manager.SignAccessToken("user-1", []string{"customer"}, 15*time.Minute)
	if err != nil {
		t.Fatalf("sign access token: %v", err)
	}

	userID, roles, err := manager.VerifyAccessToken(token)
	if err != nil {
		t.Fatalf("verify access token: %v", err)
	}

	if userID != "user-1" {
		t.Fatalf("expected user id user-1, got %q", userID)
	}
	if len(roles) != 1 || roles[0] != "customer" {
		t.Fatalf("unexpected roles: %#v", roles)
	}
}

func TestJWTManagerVerifyExpiredToken(t *testing.T) {
	manager, err := NewJWTManager("test-secret", "go-commerce")
	if err != nil {
		t.Fatalf("new jwt manager: %v", err)
	}

	claims := AccessTokenClaims{
		Roles: []string{"customer"},
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   "user-1",
			Issuer:    "go-commerce",
			IssuedAt:  jwt.NewNumericDate(time.Now().Add(-2 * time.Hour)),
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(-1 * time.Hour)),
			ID:        "expired-jti",
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	rawToken, err := token.SignedString([]byte("test-secret"))
	if err != nil {
		t.Fatalf("sign expired token: %v", err)
	}

	_, _, err = manager.VerifyAccessToken(rawToken)
	if !errors.Is(err, ErrExpiredToken) {
		t.Fatalf("expected ErrExpiredToken, got %v", err)
	}
}
