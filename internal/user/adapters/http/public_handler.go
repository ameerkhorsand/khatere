package http

import (
	"errors"
	"net/http"

	"github.com/bLorax/khatere-backend/internal/user/domain"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// PublicHandlers exposes read-only, non-sensitive user data to any
// authenticated caller — e.g. so Circle/Hangout screens can turn a
// UUID into a display name. Deliberately separate from Handlers
// (self-service /user routes): the two have different auth shapes
// and must never accidentally share a response type, or a field
// like email/bio could leak into a public response by mistake.
type PublicHandlers struct {
	users domain.UserRepository
}

func NewPublicHandlers(users domain.UserRepository) *PublicHandlers {
	return &PublicHandlers{users: users}
}

func (h *PublicHandlers) RegisterRoutes(r *gin.RouterGroup) {
	r.GET("/:id", h.GetPublicProfile)
}

// publicUserResponse is intentionally minimal. Do not add fields
// here without checking whether they are safe to expose to anyone
// with an account, not just circle connections.
type publicUserResponse struct {
	ID          string `json:"id"`
	DisplayName string `json:"display_name"`
	Handle      string `json:"handle"`
}

func (h *PublicHandlers) GetPublicProfile(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid user id"})
		return
	}

	user, err := h.users.FindByAccountID(c.Request.Context(), id)
	if err != nil {
		if errors.Is(err, domain.ErrUserNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal error"})
		return
	}

	c.JSON(http.StatusOK, publicUserResponse{
		ID:          user.AccountID.String(),
		DisplayName: user.DisplayName,
		Handle:      user.Handle,
	})
}
