package domain

import (
	"errors"
	"time"

	"github.com/google/uuid"
)

// ConnectionStatus represents the lifecycle state of a circle connection.
type ConnectionStatus string

const (
	ConnectionStatusPending  ConnectionStatus = "pending"
	ConnectionStatusAccepted ConnectionStatus = "accepted"
)

// Connection represents a relationship between two users.
//
// requester_id and addressee_id preserve the direction of a pending request.
// Once accepted, the relationship is symmetric from the caller's perspective,
// but the original direction is retained for request/history semantics.
type Connection struct {
	ID          uuid.UUID
	RequesterID uuid.UUID
	AddresseeID uuid.UUID
	Status      ConnectionStatus
	Version     int
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// IsPending reports whether the connection is awaiting acceptance.
func (c *Connection) IsPending() bool {
	return c.Status == ConnectionStatusPending
}

// IsAccepted reports whether the connection is established.
func (c *Connection) IsAccepted() bool {
	return c.Status == ConnectionStatusAccepted
}

var (
	ErrConnectionNotFound = errors.New("connection not found")
	ErrAlreadyConnected   = errors.New("already connected")
	ErrRequestNotFound    = errors.New("connection request not found")
	ErrRequestAlreadySent = errors.New("connection request already sent")
	ErrCannotConnectSelf  = errors.New("cannot connect to yourself")
	ErrBlocked            = errors.New("user is blocked")
)
