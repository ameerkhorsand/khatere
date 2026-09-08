package http

import (
	"errors"
	"net/http"

	"github.com/bLorax/khatere-backend/internal/moderator/application"
	"github.com/bLorax/khatere-backend/internal/moderator/domain"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type Handlers struct {
	createModeratorProfile *application.CreateModeratorProfileUseCase
}

func NewHandlers(createModeratorProfile *application.CreateModeratorProfileUseCase) *Handlers {
	return &Handlers{createModeratorProfile: createModeratorProfile}
}

func (h *Handlers) RegisterRoutes(r *gin.RouterGroup) {
	r.POST("/profile", h.CreateProfile)
}

func (h *Handlers) CreateProfile(c *gin.Context) {
	raw, exists := c.Get("account_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}
	accountID, err := uuid.Parse(raw.(string))
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	moderator, err := h.createModeratorProfile.Execute(c.Request.Context(), application.CreateModeratorProfileInput{
		AccountID: accountID,
	})
	if err != nil {
		if errors.Is(err, domain.ErrModeratorAlreadyExists) {
			c.JSON(http.StatusConflict, gin.H{"error": "moderator profile already exists"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal error"})
		return
	}

	c.JSON(http.StatusCreated, moderator)
}
