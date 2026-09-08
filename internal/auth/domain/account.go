package domain

import (
	"errors"
	"time"

	"github.com/google/uuid"
)

type AccountType string

const (
	AccountTypeUser      AccountType = "user"
	AccountTypeHost      AccountType = "host"
	AccountTypeModerator AccountType = "moderator"
)

func (t AccountType) Valid() bool {
	switch t {
	case AccountTypeUser, AccountTypeHost, AccountTypeModerator:
		return true
	default:
		return false
	}
}

type Account struct {
	ID           uuid.UUID
	Email        string
	PasswordHash string
	AccountType  AccountType
	Metadata     map[string]any
	Version      int
	CreatedAt    time.Time
	UpdatedAt    time.Time
	DeletedAt    *time.Time
}

func (a *Account) IsDeleted() bool {
	return a.DeletedAt != nil
}

type RefreshToken struct {
	ID        uuid.UUID
	AccountID uuid.UUID
	TokenHash string
	ExpiresAt time.Time
	Revoked   bool
	CreatedAt time.Time
}

func (r *RefreshToken) Expired() bool {
	return time.Now().After(r.ExpiresAt)
}

var (
	ErrAccountNotFound     = errors.New("account not found")
	ErrEmailAlreadyExists  = errors.New("email already registered")
	ErrInvalidCredentials  = errors.New("invalid email or password")
	ErrInvalidAccountType  = errors.New("invalid account type")
	ErrVersionConflict     = errors.New("account was modified by another request")
	ErrRefreshTokenInvalid = errors.New("refresh token invalid or expired")
	ErrRefreshTokenRevoked = errors.New("refresh token revoked")
)
