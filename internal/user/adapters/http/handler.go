package http

import (
	"errors"
	"net/http"

	"github.com/bLorax/khatere-backend/internal/user/application"
	"github.com/bLorax/khatere-backend/internal/user/domain"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type Handlers struct {
	createProfile *application.CreateProfileUseCase
	updateProfile *application.UpdateProfileUseCase
	setInterests  *application.SetInterestsUseCase
	listInterests *application.ListInterestsUseCase
	listCatalog   *application.ListInterestCatalogUseCase
}

func NewHandlers(
	createProfile *application.CreateProfileUseCase,
	updateProfile *application.UpdateProfileUseCase,
	setInterests *application.SetInterestsUseCase,
	listInterests *application.ListInterestsUseCase,
	listCatalog *application.ListInterestCatalogUseCase,
) *Handlers {
	return &Handlers{createProfile, updateProfile, setInterests, listInterests, listCatalog}
}

func (h *Handlers) RegisterRoutes(r *gin.RouterGroup) {
	r.POST("/profile", h.CreateProfile)
	r.PATCH("/profile", h.UpdateProfile)
	r.PUT("/interests", h.SetInterests)
	r.GET("/interests", h.ListInterests)
	r.GET("/interests/catalog", h.ListInterestCatalog)
}

// accountIDFromContext reads the account ID set by the auth middleware (step 7).
func accountIDFromContext(c *gin.Context) (uuid.UUID, error) {
	raw, exists := c.Get("account_id")
	if !exists {
		return uuid.UUID{}, errors.New("no account_id in context")
	}
	return uuid.Parse(raw.(string))
}

type createProfileRequest struct {
	DisplayName string  `json:"display_name" binding:"required"`
	Handle      string  `json:"handle" binding:"required"`
	Bio         *string `json:"bio"`
}

func (h *Handlers) CreateProfile(c *gin.Context) {
	accountID, err := accountIDFromContext(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	var req createProfileRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	user, err := h.createProfile.Execute(c.Request.Context(), application.CreateProfileInput{
		AccountID:   accountID,
		DisplayName: req.DisplayName,
		Handle:      req.Handle,
		Bio:         req.Bio,
	})
	if err != nil {
		switch {
		case errors.Is(err, domain.ErrUserAlreadyExists):
			c.JSON(http.StatusConflict, gin.H{"error": "profile already exists"})
		case errors.Is(err, domain.ErrHandleTaken):
			c.JSON(http.StatusConflict, gin.H{"error": "handle already taken"})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"error": "internal error"})
		}
		return
	}

	c.JSON(http.StatusCreated, user)
}

type updateProfileRequest struct {
	DisplayName       *string `json:"display_name"`
	Bio               *string `json:"bio"`
	ProfilePictureKey *string `json:"profile_picture_key"`
}

func (h *Handlers) UpdateProfile(c *gin.Context) {
	accountID, err := accountIDFromContext(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	var req updateProfileRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	user, err := h.updateProfile.Execute(c.Request.Context(), application.UpdateProfileInput{
		AccountID:         accountID,
		DisplayName:       req.DisplayName,
		Bio:               req.Bio,
		ProfilePictureKey: req.ProfilePictureKey,
	})
	if err != nil {
		if errors.Is(err, domain.ErrVersionConflict) {
			c.JSON(http.StatusConflict, gin.H{"error": "profile was updated elsewhere, please retry"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal error"})
		return
	}

	c.JSON(http.StatusOK, user)
}

type setInterestsRequest struct {
	InterestIDs []string `json:"interest_ids" binding:"required"`
}

func (h *Handlers) SetInterests(c *gin.Context) {
	accountID, err := accountIDFromContext(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	var req setInterestsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	interestIDs := make([]uuid.UUID, 0, len(req.InterestIDs))
	for _, raw := range req.InterestIDs {
		id, err := uuid.Parse(raw)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid interest id: " + raw})
			return
		}
		interestIDs = append(interestIDs, id)
	}

	if err := h.setInterests.Execute(c.Request.Context(), application.SetInterestsInput{
		UserID:      accountID,
		InterestIDs: interestIDs,
	}); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal error"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

func (h *Handlers) ListInterests(c *gin.Context) {
	accountID, err := accountIDFromContext(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	interests, err := h.listInterests.Execute(c.Request.Context(), accountID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal error"})
		return
	}
	c.JSON(http.StatusOK, interests)
}

func (h *Handlers) ListInterestCatalog(c *gin.Context) {
	if _, err := accountIDFromContext(c); err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	interests, err := h.listCatalog.Execute(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal error"})
		return
	}
	c.JSON(http.StatusOK, interests)
}
