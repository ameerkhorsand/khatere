package domain

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// Type identifies what kind of event a Notification describes. This
// list covers every trigger point across every domain — hangout,
// archive, activity, comment, and badge — since notifications are a
// cross-cutting concern that shouldn't live inside any one of them.
type Type string

const (
	TypeHangoutInvite      Type = "hangout_invite"
	TypeInviteResponded    Type = "hangout_invite_responded"
	TypeHangoutCancelled   Type = "hangout_cancelled"
	TypeMeetupPinProposed  Type = "meetup_pin_proposed"
	TypeMeetupPinChanged   Type = "meetup_pin_changed"
	TypeMeetupPinConfirmed Type = "meetup_pin_confirmed"
	TypeUploadWindowOpened Type = "upload_window_opened"
	TypeActivityApproved   Type = "activity_approved"
	TypeActivityRejected   Type = "activity_rejected"
	TypeCommentApproved    Type = "comment_approved"
	TypeCommentRejected    Type = "comment_rejected"
	TypeBadgeEarned        Type = "badge_earned"
)

// Notification is one persisted, listable event for one recipient.
// Metadata carries whatever extra, type-specific data the frontend
// needs to render it (a hangout title, an activity name, a badge
// icon key) without a follow-up request — same reasoning as
// Suggestion.Activity in the recommendation domain.
type Notification struct {
	ID          uuid.UUID
	Type        Type
	RecipientID uuid.UUID
	ActorID     uuid.UUID
	Metadata    json.RawMessage
	ReadAt      *time.Time
	CreatedAt   time.Time
}

func (n *Notification) IsRead() bool {
	return n.ReadAt != nil
}

// Event is what gets published to Kafka and later consumed to build
// a Notification row — everything except ID and CreatedAt, which
// the consumer assigns at persist time.
type Event struct {
	Type        Type
	RecipientID uuid.UUID
	ActorID     uuid.UUID
	Metadata    json.RawMessage
}
