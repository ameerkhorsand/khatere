// Package hangoutgateway is the one place the archive module reaches
// across into the hangout module. It wraps the hangout module's own
// repository interfaces and translates their types into the small,
// decoupled shapes declared by internal/archive/domain.HangoutGateway.
package hangoutgateway

import (
	"context"
	"errors"

	archivedomain "github.com/bLorax/khatere-backend/internal/archive/domain"
	hangoutdomain "github.com/bLorax/khatere-backend/internal/hangout/domain"
	"github.com/google/uuid"
)

type Gateway struct {
	hangouts     hangoutdomain.HangoutRepository
	participants hangoutdomain.ParticipantRepository
	messages     hangoutdomain.MessageRepository
}

func New(
	hangouts hangoutdomain.HangoutRepository,
	participants hangoutdomain.ParticipantRepository,
	messages hangoutdomain.MessageRepository,
) *Gateway {
	return &Gateway{hangouts: hangouts, participants: participants, messages: messages}
}

func (g *Gateway) GetResolved(ctx context.Context, hangoutID uuid.UUID) (*archivedomain.ResolvedHangout, error) {
	h, err := g.hangouts.FindByID(ctx, hangoutID)
	if err != nil {
		return nil, err
	}

	if !h.Status.IsFinal() {
		return nil, archivedomain.ErrHangoutNotResolved
	}

	participants, err := g.participants.ListParticipants(ctx, hangoutID)
	if err != nil {
		return nil, err
	}

	ids := make([]uuid.UUID, 0, len(participants))
	for _, p := range participants {
		ids = append(ids, p.UserID)
	}

	return &archivedomain.ResolvedHangout{
		ID:             h.ID,
		Status:         archivedomain.ResolvedHangoutStatus(h.Status),
		ScheduledEndAt: h.ScheduledEndAt,
		ParticipantIDs: ids,
	}, nil
}

func (g *Gateway) ListChatLog(ctx context.Context, hangoutID uuid.UUID) ([]archivedomain.ChatLine, error) {
	// No paging limit: a full chat log is needed for the snapshot.
	// hangout_messages is expected to stay small enough per hangout
	// for this to be safe; revisit if that stops being true.
	messages, err := g.messages.ListMessages(ctx, hangoutID, hangoutdomain.MessageFilter{Limit: 100000})
	if err != nil {
		return nil, err
	}

	lines := make([]archivedomain.ChatLine, 0, len(messages))
	for _, m := range messages {
		lines = append(lines, archivedomain.ChatLine{
			SenderID:  m.SenderID,
			Content:   m.Content,
			CreatedAt: m.CreatedAt,
		})
	}
	return lines, nil
}

func (g *Gateway) IsParticipant(ctx context.Context, hangoutID, userID uuid.UUID) (bool, error) {
	_, err := g.participants.FindParticipant(ctx, hangoutID, userID)
	if err != nil {
		if errors.Is(err, hangoutdomain.ErrParticipantNotFound) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

func (g *Gateway) ListResolvedForUser(ctx context.Context, userID uuid.UUID) ([]archivedomain.ResolvedHangout, error) {
	completed := hangoutdomain.HangoutStatusCompleted
	completedHangouts, err := g.hangouts.ListByUser(ctx, userID, hangoutdomain.ListFilter{Status: &completed})
	if err != nil {
		return nil, err
	}

	cancelled := hangoutdomain.HangoutStatusCancelled
	cancelledHangouts, err := g.hangouts.ListByUser(ctx, userID, hangoutdomain.ListFilter{Status: &cancelled})
	if err != nil {
		return nil, err
	}

	all := append(completedHangouts, cancelledHangouts...)
	resolved := make([]archivedomain.ResolvedHangout, 0, len(all))
	for _, h := range all {
		participants, err := g.participants.ListParticipants(ctx, h.ID)
		if err != nil {
			return nil, err
		}
		ids := make([]uuid.UUID, 0, len(participants))
		for _, p := range participants {
			ids = append(ids, p.UserID)
		}
		resolved = append(resolved, archivedomain.ResolvedHangout{
			ID:             h.ID,
			Status:         archivedomain.ResolvedHangoutStatus(h.Status),
			ScheduledEndAt: h.ScheduledEndAt,
			ParticipantIDs: ids,
		})
	}
	return resolved, nil
}
