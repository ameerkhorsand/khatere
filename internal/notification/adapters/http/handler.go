package http

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/bLorax/khatere-backend/internal/notification/application"
	"github.com/bLorax/khatere-backend/internal/notification/domain"
)

type Handlers struct {
	listNotifications *application.ListNotificationsUseCase
	markRead          *application.MarkNotificationReadUseCase
}

func NewHandlers(listNotifications *application.ListNotificationsUseCase, markRead *application.MarkNotificationReadUseCase) *Handlers {
	return &Handlers{listNotifications: listNotifications, markRead: markRead}
}

// RegisterRoutes is meant to be mounted on its own top-level,
// authenticated group in server.go (router.Group("/notifications")),
// same as recommendationGroup — notifications are always for the
// logged-in caller, never looked up by id for someone else.
// Produces GET /notifications and POST /notifications/:id/read.
func (h *Handlers) RegisterRoutes(r *gin.RouterGroup) {
	r.GET("", h.ListNotifications)
	r.POST("/:id/read", h.MarkRead)
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

// ListNotifications handles GET /notifications?limit=20. limit is
// optional; an invalid or missing value falls back to
// application.DefaultLimit inside the use case, so this handler
// doesn't need its own default.
func (h *Handlers) ListNotifications(c *gin.Context) {
	recipientID, err := accountIDFromContext(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	limit := 0
	if raw := c.Query("limit"); raw != "" {
		if parsed, err := strconv.Atoi(raw); err == nil {
			limit = parsed
		}
	}

	notifications, err := h.listNotifications.Execute(c.Request.Context(), recipientID, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal error"})
		return
	}

	if notifications == nil {
		notifications = []domain.Notification{}
	}

	c.JSON(http.StatusOK, gin.H{"notifications": notifications})
}

// MarkRead handles POST /notifications/:id/read. The repository's
// MarkRead (Step 6) scopes the update to the caller's own
// recipient_id, so this always returns 200 — whether the id
// belonged to someone else, was already read, or didn't exist, the
// caller's own notifications are left exactly as they were.
func (h *Handlers) MarkRead(c *gin.Context) {
	recipientID, err := accountIDFromContext(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	notificationID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid notification id"})
		return
	}

	if err := h.markRead.Execute(c.Request.Context(), notificationID, recipientID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal error"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "read"})
}
