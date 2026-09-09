package application

import (
	"context"
	"time"

	"github.com/bLorax/khatere-backend/internal/comment/domain"
	"github.com/google/uuid"
)

type CreateCommentUseCase struct {
	comments domain.CommentRepository
}

func NewCreateCommentUseCase(comments domain.CommentRepository) *CreateCommentUseCase {
	return &CreateCommentUseCase{comments: comments}
}

type CreateCommentInput struct {
	ActivityID uuid.UUID
	UserID     uuid.UUID
	Body       string
}

// Execute creates a comment. Any user may comment — no attendee
// check, unlike CreateRatingUseCase. Every new comment starts
// 'pending' and only becomes visible once a moderator approves it.
func (uc *CreateCommentUseCase) Execute(ctx context.Context, in CreateCommentInput) (*domain.Comment, error) {
	if in.Body == "" {
		return nil, domain.ErrEmptyBody
	}

	now := time.Now()
	comment := &domain.Comment{
		ID:         uuid.New(),
		ActivityID: in.ActivityID,
		UserID:     in.UserID,
		Body:       in.Body,
		Status:     domain.CommentStatusPending,
		Metadata:   map[string]any{},
		Version:    1,
		CreatedAt:  now,
		UpdatedAt:  now,
	}

	if err := uc.comments.Create(ctx, comment); err != nil {
		return nil, err
	}

	return comment, nil
}
