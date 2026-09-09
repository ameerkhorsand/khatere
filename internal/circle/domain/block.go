package domain

import (
	"errors"
	"time"

	"github.com/google/uuid"
)

// Block represents a directional block: BlockerID prevents interaction with
// BlockedID. Blocks are independent from connection state.
type Block struct {
	ID        uuid.UUID
	BlockerID uuid.UUID
	BlockedID uuid.UUID
	CreatedAt time.Time
}

var (
	ErrBlockNotFound   = errors.New("block not found")
	ErrAlreadyBlocked  = errors.New("user is already blocked")
	ErrCannotBlockSelf = errors.New("cannot block yourself")
)
