package http

import (
	"context"
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/bLorax/khatere-backend/internal/archive/application"
	"github.com/bLorax/khatere-backend/internal/archive/domain"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type Handlers struct {
	listArchives       *application.ListArchivesUseCase
	getArchive         *application.GetArchiveUseCase
	uploadMedia        *application.UploadMediaUseCase
	deleteArchive      *application.DeleteArchiveUseCase
	listPendingPrompts *application.ListPendingUploadPromptsUseCase
	storage            domain.MediaStorage
}

func NewHandlers(
	listArchives *application.ListArchivesUseCase,
	getArchive *application.GetArchiveUseCase,
	uploadMedia *application.UploadMediaUseCase,
	deleteArchive *application.DeleteArchiveUseCase,
	listPendingPrompts *application.ListPendingUploadPromptsUseCase,
	storage domain.MediaStorage,
) *Handlers {
	return &Handlers{
		listArchives:       listArchives,
		getArchive:         getArchive,
		uploadMedia:        uploadMedia,
		deleteArchive:      deleteArchive,
		listPendingPrompts: listPendingPrompts,
		storage:            storage,
	}
}

// mediaResponse is what the frontend receives for one ArchiveMedia
// record. StorageKey never leaves this service — it is an internal
// MinIO object key, not something the frontend should see or use.
type mediaResponse struct {
	ID              uuid.UUID `json:"ID"`
	ArchiveID       uuid.UUID `json:"ArchiveID"`
	UploaderID      uuid.UUID `json:"UploaderID"`
	MediaType       string    `json:"MediaType"`
	URL             string    `json:"URL"`
	DurationSeconds *int      `json:"DurationSeconds,omitempty"`
	CreatedAt       time.Time `json:"CreatedAt"`
}

type archiveDetailResponse struct {
	Archive domain.Archive  `json:"Archive"`
	Media   []mediaResponse `json:"Media"`
}

// toMediaResponse converts one domain record into the shape the
// frontend expects, resolving StorageKey into a fresh URL.
func toMediaResponse(ctx context.Context, storage domain.MediaStorage, m domain.ArchiveMedia) (mediaResponse, error) {
	url, err := storage.PublicURL(ctx, m.StorageKey)
	if err != nil {
		return mediaResponse{}, err
	}
	return mediaResponse{
		ID:              m.ID,
		ArchiveID:       m.ArchiveID,
		UploaderID:      m.UploaderID,
		MediaType:       string(m.MediaType),
		URL:             url,
		DurationSeconds: m.DurationSeconds,
		CreatedAt:       m.CreatedAt,
	}, nil
}

// toMediaResponseList converts a slice, stopping at the first error.
func toMediaResponseList(ctx context.Context, storage domain.MediaStorage, media []domain.ArchiveMedia) ([]mediaResponse, error) {
	out := make([]mediaResponse, 0, len(media))
	for _, m := range media {
		r, err := toMediaResponse(ctx, storage, m)
		if err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, nil
}

// RegisterRoutes mounts the archive endpoints under the given group,
// expected to be "/archives" (see cmd/api/server.go, Step 11).
func (h *Handlers) RegisterRoutes(r *gin.RouterGroup) {
	r.GET("", h.ListArchives)
	r.GET("/:id", h.GetArchive)
	r.POST("/:id/media", h.UploadMedia)
	r.POST("/:id/delete", h.DeleteArchive)
}

// RegisterUploadPromptRoute mounts the upload-prompt check onto a
// group owned by the hangout module (see cmd/api/server.go, Step 11:
// hangoutGroup.GET("/pending-upload-prompt", ...)). It lives here,
// not in the hangout module, because the logic it calls belongs to
// the archive module.
func (h *Handlers) RegisterUploadPromptRoute(r *gin.RouterGroup) {
	r.GET("/pending-upload-prompt", h.ListPendingUploadPrompts)
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

func archiveIDParam(c *gin.Context) (uuid.UUID, bool) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid archive id"})
		return uuid.UUID{}, false
	}
	return id, true
}

// handleUseCaseError maps a domain error to the right HTTP status,
// same pattern as internal/hangout/adapters/http/handler.go.
func handleUseCaseError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, domain.ErrArchiveNotFound):
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
	case errors.Is(err, domain.ErrNotArchiveParticipant):
		c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
	case errors.Is(err, domain.ErrArchiveAlreadyExists),
		errors.Is(err, domain.ErrArchiveIsPurged),
		errors.Is(err, domain.ErrAlreadyMarkedDeleted):
		c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
	case errors.Is(err, domain.ErrHangoutNotResolved),
		errors.Is(err, domain.ErrInvalidMediaType),
		errors.Is(err, domain.ErrInvalidMediaDuration),
		errors.Is(err, domain.ErrMediaTooLong):
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
	default:
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal error"})
	}
}

// -----------------------------------------------------------------
// List / Get
// -----------------------------------------------------------------

func (h *Handlers) ListArchives(c *gin.Context) {
	userID, err := accountIDFromContext(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	archives, err := h.listArchives.Execute(c.Request.Context(), application.ListArchivesInput{UserID: userID})
	if err != nil {
		handleUseCaseError(c, err)
		return
	}

	c.JSON(http.StatusOK, archives)
}

func (h *Handlers) GetArchive(c *gin.Context) {
	id, ok := archiveIDParam(c)
	if !ok {
		return
	}
	requesterID, err := accountIDFromContext(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	detail, err := h.getArchive.Execute(c.Request.Context(), application.GetArchiveInput{
		ArchiveID:   id,
		RequesterID: requesterID,
	})
	if err != nil {
		handleUseCaseError(c, err)
		return
	}

	mediaResp, err := toMediaResponseList(c.Request.Context(), h.storage, detail.Media)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal error"})
		return
	}

	c.JSON(http.StatusOK, archiveDetailResponse{Archive: detail.Archive, Media: mediaResp})
}

// -----------------------------------------------------------------
// Upload media
// -----------------------------------------------------------------

// UploadMedia expects multipart/form-data:
//   - file:             the media file
//   - media_type:       "photo" | "gif" | "video" | "audio"
//   - duration_seconds: required for video/audio, omitted otherwise
func (h *Handlers) UploadMedia(c *gin.Context) {
	archiveID, ok := archiveIDParam(c)
	if !ok {
		return
	}
	uploaderID, err := accountIDFromContext(c)
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

	mediaType := domain.MediaType(c.PostForm("media_type"))

	var duration *int
	if raw := c.PostForm("duration_seconds"); raw != "" {
		d, err := strconv.Atoi(raw)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid duration_seconds"})
			return
		}
		duration = &d
	}

	media, err := h.uploadMedia.Execute(c.Request.Context(), application.UploadMediaInput{
		ArchiveID:       archiveID,
		UploaderID:      uploaderID,
		MediaType:       mediaType,
		Filename:        fileHeader.Filename,
		Content:         file,
		SizeBytes:       fileHeader.Size,
		DurationSeconds: duration,
	})
	if err != nil {
		handleUseCaseError(c, err)
		return
	}

	resp, err := toMediaResponse(c.Request.Context(), h.storage, *media)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal error"})
		return
	}

	c.JSON(http.StatusCreated, resp)
}

// -----------------------------------------------------------------
// Personal delete
// -----------------------------------------------------------------

func (h *Handlers) DeleteArchive(c *gin.Context) {
	id, ok := archiveIDParam(c)
	if !ok {
		return
	}
	userID, err := accountIDFromContext(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	purged, err := h.deleteArchive.Execute(c.Request.Context(), application.DeleteArchiveInput{
		ArchiveID: id,
		UserID:    userID,
	})
	if err != nil {
		handleUseCaseError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"purged": purged})
}

// -----------------------------------------------------------------
// Upload prompt
// -----------------------------------------------------------------

func (h *Handlers) ListPendingUploadPrompts(c *gin.Context) {
	userID, err := accountIDFromContext(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	prompts, err := h.listPendingPrompts.Execute(c.Request.Context(), application.ListPendingUploadPromptsInput{UserID: userID})
	if err != nil {
		handleUseCaseError(c, err)
		return
	}

	c.JSON(http.StatusOK, prompts)
}
