package http

import (
	"errors"
	"net/http"

	"github.com/bLorax/khatere-backend/internal/userbadge/application"
	"github.com/bLorax/khatere-backend/internal/userbadge/domain"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type Handlers struct {
	listUserBadges *application.ListUserBadgesUseCase
}

func NewHandlers(listUserBadges *application.ListUserBadgesUseCase) *Handlers {
	return &Handlers{listUserBadges: listUserBadges}
}

// RegisterRoutes is mounted under the authenticated /user group in
// server.go. This always returns the caller's own badges, so it
// needs no id in the URL.
func (h *Handlers) RegisterRoutes(r *gin.RouterGroup) {
	r.GET("/badges", h.ListMyBadges)
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

func (h *Handlers) ListMyBadges(c *gin.Context) {
	userID, err := accountIDFromContext(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	badges, err := h.listUserBadges.Execute(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal error"})
		return
	}

	if badges == nil {
		badges = []domain.UserBadge{}
	}

	c.JSON(http.StatusOK, badges)
}
