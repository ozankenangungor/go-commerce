package auth

import (
	"fmt"

	"golang.org/x/crypto/bcrypt"
)

// BcryptHasher provides password hashing and verification.
type BcryptHasher struct {
	cost int
}

// NewBcryptHasher creates a bcrypt hasher with the provided cost.
func NewBcryptHasher(cost int) (*BcryptHasher, error) {
	if cost < bcrypt.MinCost || cost > bcrypt.MaxCost {
		return nil, fmt.Errorf("bcrypt cost must be between %d and %d", bcrypt.MinCost, bcrypt.MaxCost)
	}
	return &BcryptHasher{cost: cost}, nil
}

// Hash hashes a plaintext password.
func (h *BcryptHasher) Hash(password string) (string, error) {
	hashBytes, err := bcrypt.GenerateFromPassword([]byte(password), h.cost)
	if err != nil {
		return "", fmt.Errorf("hash password: %w", err)
	}
	return string(hashBytes), nil
}

// Verify compares a stored hash with a candidate plaintext password.
func (h *BcryptHasher) Verify(hash string, password string) error {
	if err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)); err != nil {
		return err
	}
	return nil
}
