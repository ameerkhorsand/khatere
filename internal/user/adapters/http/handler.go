package http

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/bLorax/khatere-backend/internal/user/application"
	"github.com/bLorax/khatere-backend/internal/user/domain"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type Handlers struct {
	createProfile        *application.CreateProfileUseCase
	getProfile           *application.GetProfileUseCase
	updateProfile        *application.UpdateProfileUseCase
	setInterests         *application.SetInterestsUseCase
	listInterests        *application.ListInterestsUseCase
	listCatalog          *application.ListInterestCatalogUseCase
	uploadProfilePicture *application.UploadProfilePictureUseCase
	storage              domain.ProfileStorage
}

func NewHandlers(
	createProfile *application.CreateProfileUseCase,
	getProfile *application.GetProfileUseCase,
	updateProfile *application.UpdateProfileUseCase,
	setInterests *application.SetInterestsUseCase,
	listInterests *application.ListInterestsUseCase,
	listCatalog *application.ListInterestCatalogUseCase,
	uploadProfilePicture *application.UploadProfilePictureUseCase,
	storage domain.ProfileStorage,
) *Handlers {
	return &Handlers{
		createProfile:        createProfile,
		getProfile:           getProfile,
		updateProfile:        updateProfile,
		setInterests:         setInterests,
		listInterests:        listInterests,
		listCatalog:          listCatalog,
		uploadProfilePicture: uploadProfilePicture,
		storage:              storage,
	}
}

// profileResponse is what the frontend receives for a user profile.
// ProfilePictureKey never leaves this service — it is an internal
// MinIO object key, not something the frontend should see or use.
// Mirrors archive's mediaResponse / toMediaResponse pattern.
type profileResponse struct {
	AccountID   uuid.UUID      `json:"AccountID"`
	DisplayName string         `json:"DisplayName"`
	Handle      string         `json:"Handle"`
	Bio         *string        `json:"Bio"`
	URL         *string        `json:"URL,omitempty"`
	Metadata    map[string]any `json:"Metadata"`
	Version     int            `json:"Version"`
	CreatedAt   time.Time      `json:"CreatedAt"`
	UpdatedAt   time.Time      `json:"UpdatedAt"`
}

// toProfileResponse converts a domain.User into the shape the
// frontend expects, resolving ProfilePictureKey into a fresh URL.
// If the user has no picture, URL is left nil.
func toProfileResponse(ctx context.Context, storage domain.ProfileStorage, u *domain.User) (profileResponse, error) {
	resp := profileResponse{
		AccountID:   u.AccountID,
		DisplayName: u.DisplayName,
		Handle:      u.Handle,
		Bio:         u.Bio,
		Metadata:    u.Metadata,
		Version:     u.Version,
		CreatedAt:   u.CreatedAt,
		UpdatedAt:   u.UpdatedAt,
	}

	if u.ProfilePictureKey != nil {
		url, err := storage.PublicURL(ctx, *u.ProfilePictureKey)
		if err != nil {
			return profileResponse{}, err
		}
		resp.URL = &url
	}

	return resp, nil
}

func (h *Handlers) RegisterRoutes(r *gin.RouterGroup) {
	r.POST("/profile", h.CreateProfile)
	r.GET("/profile", h.GetProfile)
	r.PATCH("/profile", h.UpdateProfile)
	r.POST("/profile/picture", h.UploadProfilePicture)
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

	resp, err := toProfileResponse(c.Request.Context(), h.storage, user)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal error"})
		return
	}

	c.JSON(http.StatusCreated, resp)
}

func (h *Handlers) GetProfile(c *gin.Context) {
	accountID, err := accountIDFromContext(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	user, err := h.getProfile.Execute(c.Request.Context(), accountID)
	if err != nil {
		if errors.Is(err, domain.ErrUserNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "user profile not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal error"})
		return
	}

	resp, err := toProfileResponse(c.Request.Context(), h.storage, user)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal error"})
		return
	}

	c.JSON(http.StatusOK, resp)
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

	resp, err := toProfileResponse(c.Request.Context(), h.storage, user)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal error"})
		return
	}

	c.JSON(http.StatusOK, resp)
}

// UploadProfilePicture reads a multipart file (field name "file",
// same as archive's UploadMedia), stores it, and returns the
// updated profile with a resolved URL.
func (h *Handlers) UploadProfilePicture(c *gin.Context) {
	accountID, err := accountIDFromContext(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	fileHeader, err := c.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "missing file"})
		return
	}
	file, err := fileHeader.Open()
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "could not read file"})
		return
	}
	defer file.Close()

	user, err := h.uploadProfilePicture.Execute(c.Request.Context(), application.UploadProfilePictureInput{
		AccountID:   accountID,
		Filename:    fileHeader.Filename,
		ContentType: fileHeader.Header.Get("Content-Type"),
		Content:     file,
		SizeBytes:   fileHeader.Size,
	})
	if err != nil {
		switch {
		case errors.Is(err, domain.ErrInvalidProfilePictureType):
			c.JSON(http.StatusBadRequest, gin.H{"error": "profile picture must be jpg, png, or webp"})
		case errors.Is(err, domain.ErrProfilePictureTooLarge):
			c.JSON(http.StatusBadRequest, gin.H{"error": "profile picture exceeds the 10 MB limit"})
		case errors.Is(err, domain.ErrUserNotFound):
			c.JSON(http.StatusNotFound, gin.H{"error": "user profile not found"})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"error": "internal error"})
		}
		return
	}

	resp, err := toProfileResponse(c.Request.Context(), h.storage, user)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal error"})
		return
	}

	c.JSON(http.StatusOK, resp)
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
