package service

import "errors"

const (
	CodeAuthInvalidCredentials  = "AUTH_INVALID_CREDENTIALS"
	CodeAuthEmailTaken          = "AUTH_EMAIL_TAKEN"
	CodeAuthInvalidToken        = "AUTH_INVALID_TOKEN"
	CodeAuthExpiredToken        = "AUTH_EXPIRED_TOKEN"
	CodeAuthInvalidRefreshToken = "AUTH_INVALID_REFRESH_TOKEN"
)

var (
	// ErrUnavailable marks dependency/infra availability failures.
	ErrUnavailable = errors.New("service dependency unavailable")
)

// DomainError represents expected business/auth failures returned in protobuf envelopes.
type DomainError struct {
	Code    string
	Message string
}

func (e *DomainError) Error() string {
	if e == nil {
		return "domain error"
	}
	if e.Message == "" {
		return e.Code
	}
	return e.Code + ": " + e.Message
}
