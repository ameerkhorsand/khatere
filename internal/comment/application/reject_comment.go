package application

import (
	"context"
	"encoding/json"
	"log"
	"time"

	"github.com/bLorax/khatere-backend/internal/comment/domain"
	notificationdomain "github.com/bLorax/khatere-backend/internal/notification/domain"
	"github.com/google/uuid"
)

type RejectCommentUseCase struct {
	comments domain.CommentRepository
	notify   NotificationPublisher
}

func NewRejectCommentUseCase(comments domain.CommentRepository, notify NotificationPublisher) *RejectCommentUseCase {
	return &RejectCommentUseCase{comments: comments, notify: notify}
}

type RejectCommentInput struct {
	CommentID   uuid.UUID
	ModeratorID uuid.UUID
}

func (uc *RejectCommentUseCase) Execute(ctx context.Context, in RejectCommentInput) (*domain.Comment, error) {
	comment, err := uc.comments.FindByID(ctx, in.CommentID)
	if err != nil {
		return nil, err
	}

	if comment.Status != domain.CommentStatusPending {
		return nil, domain.ErrCommentNotPending
	}

	now := time.Now()
	comment.Status = domain.CommentStatusRejected
	comment.ReviewedBy = &in.ModeratorID
	comment.ReviewedAt = &now

	if err := uc.comments.Update(ctx, comment); err != nil {
		return nil, err
	}

	metadata, _ := json.Marshal(map[string]string{
		"comment_id":  comment.ID.String(),
		"activity_id": comment.ActivityID.String(),
	})
	if err := uc.notify.Publish(ctx, notificationdomain.Event{
		Type:        notificationdomain.TypeCommentRejected,
		RecipientID: comment.UserID,
		ActorID:     in.ModeratorID,
		Metadata:    metadata,
	}); err != nil {
		log.Printf("comment: failed to publish rejection notification for %s: %v", comment.ID, err)
	}

	return comment, nil
}
