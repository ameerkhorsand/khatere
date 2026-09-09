package domain

import (
	"errors"
	"time"

	"github.com/google/uuid"
)

// -----------------------------------------------------------------
// Comment
// -----------------------------------------------------------------

type CommentStatus string

const (
	CommentStatusPending  CommentStatus = "pending"
	CommentStatusApproved CommentStatus = "approved"
	CommentStatusRejected CommentStatus = "rejected"
)

func (s CommentStatus) Valid() bool {
	switch s {
	case CommentStatusPending, CommentStatusApproved, CommentStatusRejected:
		return true
	default:
		return false
	}
}

type Comment struct {
	ID         uuid.UUID
	ActivityID uuid.UUID
	UserID     uuid.UUID
	Body       string
	Status     CommentStatus
	ReviewedBy *uuid.UUID
	ReviewedAt *time.Time
	Metadata   map[string]any
	Version    int
	CreatedAt  time.Time
	UpdatedAt  time.Time
	DeletedAt  *time.Time
}

func (c *Comment) IsDeleted() bool {
	return c.DeletedAt != nil
}

func (c *Comment) IsVisible() bool {
	return c.Status == CommentStatusApproved && !c.IsDeleted()
}

// -----------------------------------------------------------------
// CommentVote
// -----------------------------------------------------------------

type VoteValue int

const (
	VoteUp   VoteValue = 1
	VoteDown VoteValue = -1
)

func (v VoteValue) Valid() bool {
	return v == VoteUp || v == VoteDown
}

type CommentVote struct {
	CommentID uuid.UUID
	UserID    uuid.UUID
	Value     VoteValue
	CreatedAt time.Time
}

// -----------------------------------------------------------------
// Domain errors
// -----------------------------------------------------------------

var (
	ErrCommentNotFound   = errors.New("comment not found")
	ErrEmptyBody         = errors.New("comment body must not be empty")
	ErrCommentNotPending = errors.New("comment is not pending review")
	ErrVersionConflict   = errors.New("comment was modified by another request")
	ErrInvalidVoteValue  = errors.New("vote value must be up or down")
)
