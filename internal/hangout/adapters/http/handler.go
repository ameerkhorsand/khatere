package http

import (
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/bLorax/khatere-backend/internal/hangout/application"
	"github.com/bLorax/khatere-backend/internal/hangout/domain"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type Handlers struct {
	createHangout      *application.CreateHangoutUseCase
	getHangout         *application.GetHangoutUseCase
	listHangouts       *application.ListHangoutsUseCase
	inviteParticipants *application.InviteParticipantsUseCase
	respondToInvite    *application.RespondToInviteUseCase
	cancelHangout      *application.CancelHangoutUseCase
	updateStatus       *application.UpdateHangoutStatusUseCase
	sendMessage        *application.SendMessageUseCase
	listMessages       *application.ListMessagesUseCase
	proposeMeetupPin   *application.ProposeMeetupPinUseCase
	respondToMeetupPin *application.RespondToMeetupPinUseCase
	getMeetupPin       *application.GetMeetupPinUseCase
}

func NewHandlers(
	createHangout *application.CreateHangoutUseCase,
	getHangout *application.GetHangoutUseCase,
	listHangouts *application.ListHangoutsUseCase,
	inviteParticipants *application.InviteParticipantsUseCase,
	respondToInvite *application.RespondToInviteUseCase,
	cancelHangout *application.CancelHangoutUseCase,
	updateStatus *application.UpdateHangoutStatusUseCase,
	sendMessage *application.SendMessageUseCase,
	listMessages *application.ListMessagesUseCase,
	proposeMeetupPin *application.ProposeMeetupPinUseCase,
	respondToMeetupPin *application.RespondToMeetupPinUseCase,
	getMeetupPin *application.GetMeetupPinUseCase,
) *Handlers {
	return &Handlers{
		createHangout, getHangout, listHangouts,
		inviteParticipants, respondToInvite, cancelHangout,
		updateStatus, sendMessage, listMessages,
		proposeMeetupPin, respondToMeetupPin, getMeetupPin,
	}
}

func (h *Handlers) RegisterRoutes(r *gin.RouterGroup) {
	r.POST("", h.CreateHangout)
	r.GET("", h.ListHangouts)
	r.GET("/:id", h.GetHangout)

	r.POST("/:id/invites", h.InviteParticipants)
	r.POST("/:id/invites/respond", h.RespondToInvite)

	r.POST("/:id/cancel", h.CancelHangout)
	r.POST("/:id/status", h.UpdateStatus)

	r.GET("/:id/messages", h.ListMessages)
	r.POST("/:id/messages", h.SendMessage)

	r.GET("/:id/pin", h.GetMeetupPin)
	r.POST("/:id/pin", h.ProposeMeetupPin)
	r.POST("/:id/pin/respond", h.RespondToMeetupPin)
}

func accountIDFromContext(c *gin.Context) (uuid.UUID, error) {
	raw, exists := c.Get("account_id")
	if !exists {
		return uuid.UUID{}, errors.New("no account_id in context")
	}
	id, err := uuid.Parse(raw.(string))
	if err != nil {
		return uuid.UUID{}, errors.New("invalid account_id in context")
	}
	return id, nil
}

// hangoutIDParam reads and parses the :id route parameter shared by
// every handler below.
func hangoutIDParam(c *gin.Context) (uuid.UUID, bool) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid hangout id"})
		return uuid.UUID{}, false
	}
	return id, true
}

// handleUseCaseError maps a domain error to the right HTTP status.
// Every handler below funnels its use-case error through here, so
// the mapping stays in one place instead of repeated per handler.
func handleUseCaseError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, domain.ErrHangoutNotFound),
		errors.Is(err, domain.ErrParticipantNotFound),
		errors.Is(err, domain.ErrPinNotFound),
		errors.Is(err, domain.ErrPinConfirmationNotFound):
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
	case errors.Is(err, domain.ErrNotOrganizer),
		errors.Is(err, domain.ErrNotInvited),
		errors.Is(err, domain.ErrNotAcceptedParticipant):
		c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
	case errors.Is(err, domain.ErrHangoutIsFinal),
		errors.Is(err, domain.ErrVersionConflict),
		errors.Is(err, domain.ErrParticipantLimit),
		errors.Is(err, domain.ErrAlreadyParticipant),
		errors.Is(err, domain.ErrInviteAlreadyAnswered),
		errors.Is(err, domain.ErrPinVersionConflict):
		c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
	case errors.Is(err, domain.ErrInvalidHangoutStatus),
		errors.Is(err, application.ErrEmptyMessage),
		errors.Is(err, application.ErrMessageTooLong):
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
	default:
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal error"})
	}
}

// -----------------------------------------------------------------
// Create / Get / List
// -----------------------------------------------------------------

type createHangoutRequest struct {
	Title       string     `json:"title" binding:"required"`
	Description *string    `json:"description"`
	ScheduledAt *time.Time `json:"scheduled_at"`
	// ActivityID links this hangout to an activity, so a later
	// completed status lets attendees rate/comment on that
	// activity (see rating.AttendanceChecker). Nil for a plain
	// user-organized hangout with no linked activity.
	ActivityID *uuid.UUID `json:"activity_id"`
}

func (h *Handlers) CreateHangout(c *gin.Context) {
	organizerID, err := accountIDFromContext(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	var req createHangoutRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	hangout, err := h.createHangout.Execute(c.Request.Context(), application.CreateHangoutInput{
		OrganizerID: organizerID,
		Title:       req.Title,
		Description: req.Description,
		ScheduledAt: req.ScheduledAt,
	})
	if err != nil {
		handleUseCaseError(c, err)
		return
	}

	c.JSON(http.StatusCreated, hangout)
}

func (h *Handlers) GetHangout(c *gin.Context) {
	id, ok := hangoutIDParam(c)
	if !ok {
		return
	}
	requesterID, err := accountIDFromContext(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	detail, err := h.getHangout.Execute(c.Request.Context(), application.GetHangoutInput{
		HangoutID:   id,
		RequesterID: requesterID,
	})
	if err != nil {
		handleUseCaseError(c, err)
		return
	}

	c.JSON(http.StatusOK, detail)
}

func (h *Handlers) ListHangouts(c *gin.Context) {
	userID, err := accountIDFromContext(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	var status *domain.HangoutStatus
	if raw := c.Query("status"); raw != "" {
		s := domain.HangoutStatus(raw)
		if !s.Valid() {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid status filter"})
			return
		}
		status = &s
	}

	hangouts, err := h.listHangouts.Execute(c.Request.Context(), application.ListHangoutsInput{
		UserID: userID,
		Status: status,
	})
	if err != nil {
		handleUseCaseError(c, err)
		return
	}

	c.JSON(http.StatusOK, hangouts)
}

// -----------------------------------------------------------------
// Invites
// -----------------------------------------------------------------

type inviteParticipantsRequest struct {
	UserIDs []uuid.UUID `json:"user_ids" binding:"required,min=1"`
}

func (h *Handlers) InviteParticipants(c *gin.Context) {
	id, ok := hangoutIDParam(c)
	if !ok {
		return
	}
	inviterID, err := accountIDFromContext(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	var req inviteParticipantsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	results, err := h.inviteParticipants.Execute(c.Request.Context(), application.InviteParticipantsInput{
		HangoutID:  id,
		InviterID:  inviterID,
		InviteeIDs: req.UserIDs,
	})
	if err != nil {
		// This is a whole-request failure (bad hangout, not the
		// organizer, hangout already closed) — per-invitee failures
		// come back inside results instead, with 200 below.
		handleUseCaseError(c, err)
		return
	}

	c.JSON(http.StatusOK, formatInviteResults(results))
}

// formatInviteResults turns each InviteResult into plain JSON: the
// participant on success, or an error string on failure, so the
// caller doesn't need to know about Go error values.
func formatInviteResults(results []application.InviteResult) []gin.H {
	out := make([]gin.H, 0, len(results))
	for _, r := range results {
		if r.Err != nil {
			out = append(out, gin.H{"user_id": r.UserID, "error": r.Err.Error()})
			continue
		}
		out = append(out, gin.H{"user_id": r.UserID, "participant": r.Participant})
	}
	return out
}

type respondToInviteRequest struct {
	Accept bool    `json:"accept"`
	Reason *string `json:"reason"`
}

func (h *Handlers) RespondToInvite(c *gin.Context) {
	id, ok := hangoutIDParam(c)
	if !ok {
		return
	}
	userID, err := accountIDFromContext(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	var req respondToInviteRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	participant, err := h.respondToInvite.Execute(c.Request.Context(), application.RespondToInviteInput{
		HangoutID: id,
		UserID:    userID,
		Accept:    req.Accept,
		Reason:    req.Reason,
	})
	if err != nil {
		handleUseCaseError(c, err)
		return
	}

	c.JSON(http.StatusOK, participant)
}

// -----------------------------------------------------------------
// Status changes
// -----------------------------------------------------------------

func (h *Handlers) CancelHangout(c *gin.Context) {
	id, ok := hangoutIDParam(c)
	if !ok {
		return
	}
	requesterID, err := accountIDFromContext(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	hangout, err := h.cancelHangout.Execute(c.Request.Context(), application.CancelHangoutInput{
		HangoutID:   id,
		RequesterID: requesterID,
	})
	if err != nil {
		handleUseCaseError(c, err)
		return
	}

	c.JSON(http.StatusOK, hangout)
}

type updateStatusRequest struct {
	Status domain.HangoutStatus `json:"status" binding:"required"`
}

func (h *Handlers) UpdateStatus(c *gin.Context) {
	id, ok := hangoutIDParam(c)
	if !ok {
		return
	}
	requesterID, err := accountIDFromContext(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	var req updateStatusRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	hangout, err := h.updateStatus.Execute(c.Request.Context(), application.UpdateHangoutStatusInput{
		HangoutID:   id,
		RequesterID: requesterID,
		NewStatus:   req.Status,
	})
	if err != nil {
		handleUseCaseError(c, err)
		return
	}

	c.JSON(http.StatusOK, hangout)
}

// -----------------------------------------------------------------
// Chat
// -----------------------------------------------------------------

type sendMessageRequest struct {
	Content string `json:"content" binding:"required"`
}

func (h *Handlers) SendMessage(c *gin.Context) {
	id, ok := hangoutIDParam(c)
	if !ok {
		return
	}
	senderID, err := accountIDFromContext(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	var req sendMessageRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	message, err := h.sendMessage.Execute(c.Request.Context(), application.SendMessageInput{
		HangoutID: id,
		SenderID:  senderID,
		Content:   req.Content,
	})
	if err != nil {
		handleUseCaseError(c, err)
		return
	}

	c.JSON(http.StatusCreated, message)
}

func (h *Handlers) ListMessages(c *gin.Context) {
	id, ok := hangoutIDParam(c)
	if !ok {
		return
	}
	requesterID, err := accountIDFromContext(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	var before *time.Time
	if raw := c.Query("before"); raw != "" {
		t, err := time.Parse(time.RFC3339, raw)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid before timestamp, expected RFC3339"})
			return
		}
		before = &t
	}

	limit := 0
	if raw := c.Query("limit"); raw != "" {
		n, err := strconv.Atoi(raw)
		if err != nil || n < 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid limit"})
			return
		}
		limit = n
	}

	messages, err := h.listMessages.Execute(c.Request.Context(), application.ListMessagesInput{
		HangoutID:   id,
		RequesterID: requesterID,
		Before:      before,
		Limit:       limit,
	})
	if err != nil {
		handleUseCaseError(c, err)
		return
	}

	c.JSON(http.StatusOK, messages)
}

// -----------------------------------------------------------------
// Meetup Pin
// -----------------------------------------------------------------

type proposeMeetupPinRequest struct {
	PlaceName string  `json:"place_name" binding:"required"`
	Address   *string `json:"address"`
	// Latitude and Longitude have no "required" tag on purpose:
	// Go's validator treats a numeric zero as "missing", but 0,0 is
	// a real coordinate. PlaceName being required is enough to
	// catch a genuinely empty request.
	Latitude    float64    `json:"latitude"`
	Longitude   float64    `json:"longitude"`
	ScheduledAt *time.Time `json:"scheduled_at"`
}

// ProposeMeetupPin handles both the first pin for a hangout and any
// later change to it — same request shape either way. A change
// resets every participant's confirmation, which is why the
// frontend should treat this as "propose or update", not just
// "create".
func (h *Handlers) ProposeMeetupPin(c *gin.Context) {
	id, ok := hangoutIDParam(c)
	if !ok {
		return
	}
	proposerID, err := accountIDFromContext(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	var req proposeMeetupPinRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	pin, err := h.proposeMeetupPin.Execute(c.Request.Context(), application.ProposeMeetupPinInput{
		HangoutID:   id,
		ProposerID:  proposerID,
		PlaceName:   req.PlaceName,
		Address:     req.Address,
		Latitude:    req.Latitude,
		Longitude:   req.Longitude,
		ScheduledAt: req.ScheduledAt,
	})
	if err != nil {
		handleUseCaseError(c, err)
		return
	}

	c.JSON(http.StatusOK, pin)
}

type respondToMeetupPinRequest struct {
	Confirm bool `json:"confirm"`
}

func (h *Handlers) RespondToMeetupPin(c *gin.Context) {
	id, ok := hangoutIDParam(c)
	if !ok {
		return
	}
	userID, err := accountIDFromContext(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	var req respondToMeetupPinRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	confirmation, err := h.respondToMeetupPin.Execute(c.Request.Context(), application.RespondToMeetupPinInput{
		HangoutID: id,
		UserID:    userID,
		Confirm:   req.Confirm,
	})
	if err != nil {
		handleUseCaseError(c, err)
		return
	}

	c.JSON(http.StatusOK, confirmation)
}

// GetMeetupPin returns the current pin plus every participant's
// confirmation state, for the pinned banner on the Hangout Chat
// screen. A 404 here means no pin has been proposed yet — the
// frontend should treat that as "nothing pinned", not an error.
func (h *Handlers) GetMeetupPin(c *gin.Context) {
	id, ok := hangoutIDParam(c)
	if !ok {
		return
	}
	requesterID, err := accountIDFromContext(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	summary, err := h.getMeetupPin.Execute(c.Request.Context(), application.GetMeetupPinInput{
		HangoutID:   id,
		RequesterID: requesterID,
	})
	if err != nil {
		handleUseCaseError(c, err)
		return
	}

	c.JSON(http.StatusOK, summary)
}
