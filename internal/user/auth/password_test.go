package auth

import (
	"testing"

	"golang.org/x/crypto/bcrypt"
)

func TestBcryptHasherHashAndVerify(t *testing.T) {
	hasher, err := NewBcryptHasher(bcrypt.MinCost)
	if err != nil {
		t.Fatalf("new hasher: %v", err)
	}

	password := "StrongPass123!"
	hash, err := hasher.Hash(password)
	if err != nil {
		t.Fatalf("hash password: %v", err)
	}

	tests := []struct {
		name          string
		candidate     string
		expectFailure bool
	}{
		{name: "valid password", candidate: password, expectFailure: false},
		{name: "invalid password", candidate: "wrong-password", expectFailure: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := hasher.Verify(hash, tt.candidate)
			if tt.expectFailure && err == nil {
				t.Fatal("expected verification failure")
			}
			if !tt.expectFailure && err != nil {
				t.Fatalf("expected verification success, got %v", err)
			}
		})
	}
}
