package domain

import (
	"context"

	"github.com/google/uuid"
)

// This repo has no Event/Notification infrastructure yet (checked in
// step 1). Notifier is a minimal placeholder: a stable interface the
// application layer can call today, that a real notification system
// can implement later without changing any use case.

type NotificationType string

const (
	NotificationHangoutInvite    NotificationType = "hangout_invite"
	NotificationInviteResponded  NotificationType = "hangout_invite_responded"
	NotificationHangoutCancelled NotificationType = "hangout_cancelled"

	// NotificationMeetupPinProposed fires once, when a hangout gets
	// its first pin.
	NotificationMeetupPinProposed NotificationType = "meetup_pin_proposed"
	// NotificationMeetupPinChanged fires on every later change to an
	// existing pin. It tells each participant they must confirm
	// again, since their prior confirmation was reset.
	NotificationMeetupPinChanged NotificationType = "meetup_pin_changed"
	// NotificationMeetupPinConfirmed goes to the organizer each time
	// a participant confirms the current pin.
	NotificationMeetupPinConfirmed NotificationType = "meetup_pin_confirmed"
)

// Notification describes one event worth telling someone about.
// ActorID is who caused it; RecipientID is who should hear about it.
type Notification struct {
	Type        NotificationType
	RecipientID uuid.UUID
	ActorID     uuid.UUID
	HangoutID   uuid.UUID
}

// Notifier delivers one notification. A future adapter can email it,
// push it, or write it to a notifications table — the use cases below
// only ever call Notify, so swapping the adapter needs no other changes.
type Notifier interface {
	Notify(ctx context.Context, n Notification) error
}
