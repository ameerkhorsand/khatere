package domain

import (
	"errors"
	"time"

	"github.com/google/uuid"
)

type Host struct {
	AccountID    uuid.UUID
	BusinessName string
	LocationInfo *string
	Metadata     map[string]any
	Version      int
	CreatedAt    time.Time
	UpdatedAt    time.Time
	DeletedAt    *time.Time
}

func (h *Host) IsDeleted() bool {
	return h.DeletedAt != nil
}

var (
	ErrHostNotFound      = errors.New("host not found")
	ErrHostAlreadyExists = errors.New("host profile already exists")
	ErrVersionConflict   = errors.New("host was modified by another request")
)
