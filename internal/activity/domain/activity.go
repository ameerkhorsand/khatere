package domain

import (
	"errors"
	"time"

	"github.com/google/uuid"
)

type SourceType string

const (
	SourceTypeModerator SourceType = "moderator"
	SourceTypeHost      SourceType = "host"
	SourceTypeUser      SourceType = "user"
)

func (s SourceType) Valid() bool {
	switch s {
	case SourceTypeModerator, SourceTypeHost, SourceTypeUser:
		return true
	default:
		return false
	}
}

type ActivityStatus string

const (
	ActivityStatusPending  ActivityStatus = "pending"
	ActivityStatusApproved ActivityStatus = "approved"
	ActivityStatusRejected ActivityStatus = "rejected"
)

type Activity struct {
	ID              uuid.UUID
	Title           string
	Description     *string
	SourceType      SourceType
	CreatedBy       uuid.UUID
	Status          ActivityStatus
	ReviewedBy      *uuid.UUID
	ReviewedAt      *time.Time
	RejectionReason *string
	Metadata        map[string]any
	Version         int
	CreatedAt       time.Time
	UpdatedAt       time.Time
	DeletedAt       *time.Time
}

func (a *Activity) IsDeleted() bool {
	return a.DeletedAt != nil
}

func (a *Activity) IsVisible() bool {
	return a.Status == ActivityStatusApproved && !a.IsDeleted()
}

var (
	ErrActivityNotFound      = errors.New("activity not found")
	ErrInvalidSourceType     = errors.New("invalid source type")
	ErrActivityNotPending    = errors.New("activity is not pending review")
	ErrVersionConflict       = errors.New("activity was modified by another request")
	ErrNotAuthorizedToReview = errors.New("only moderators can review activities")
)
