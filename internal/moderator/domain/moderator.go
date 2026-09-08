package domain

import (
	"errors"
	"time"

	"github.com/google/uuid"
)

type Moderator struct {
	AccountID uuid.UUID
	Metadata  map[string]any
	Version   int
	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt *time.Time
}

func (m *Moderator) IsDeleted() bool {
	return m.DeletedAt != nil
}

var (
	ErrModeratorNotFound      = errors.New("moderator not found")
	ErrModeratorAlreadyExists = errors.New("moderator profile already exists")
	ErrVersionConflict        = errors.New("moderator was modified by another request")
)
