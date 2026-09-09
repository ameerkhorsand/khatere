package domain

import (
	"context"

	"github.com/google/uuid"
)

// ConnectionRepository persists circle connection requests and accepted
// connections. RequesterID/AddresseeID preserve request direction.
type ConnectionRepository interface {
	CreateRequest(ctx context.Context, requesterID, addresseeID uuid.UUID) (*Connection, error)
	Accept(ctx context.Context, requestID, addresseeID uuid.UUID) (*Connection, error)
	Decline(ctx context.Context, requestID, addresseeID uuid.UUID) error
	Sever(ctx context.Context, connectionID, userID uuid.UUID) error

	ListCircle(ctx context.Context, userID uuid.UUID) ([]Connection, error)
	ListIncomingPending(ctx context.Context, userID uuid.UUID) ([]Connection, error)
	ListOutgoingPending(ctx context.Context, userID uuid.UUID) ([]Connection, error)
}

// BlockRepository persists directional user blocks.
type BlockRepository interface {
	Block(ctx context.Context, blockerID, blockedID uuid.UUID) (*Block, error)
	Unblock(ctx context.Context, blockerID, blockedID uuid.UUID) error
	List(ctx context.Context, blockerID uuid.UUID) ([]Block, error)
	IsBlocked(ctx context.Context, blockerID, blockedID uuid.UUID) (bool, error)
}
