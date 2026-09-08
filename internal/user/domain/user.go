package domain

import (
	"errors"
	"time"

	"github.com/google/uuid"
)

type User struct {
	AccountID         uuid.UUID
	DisplayName       string
	Bio               *string
	ProfilePictureKey *string // object storage key, not a URL — generate a presigned URL on read
	Metadata          map[string]any
	Version           int
	CreatedAt         time.Time
	UpdatedAt         time.Time
	DeletedAt         *time.Time
}

func (u *User) IsDeleted() bool {
	return u.DeletedAt != nil
}

type Interest struct {
	ID        uuid.UUID
	Slug      string
	Label     string
	Category  *string
	Active    bool
	SortOrder int
	CreatedAt time.Time
}

var (
	ErrUserNotFound      = errors.New("user not found")
	ErrUserAlreadyExists = errors.New("user profile already exists")
	ErrVersionConflict   = errors.New("user was modified by another request")
)
