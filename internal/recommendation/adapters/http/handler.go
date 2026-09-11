package http

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/bLorax/khatere-backend/internal/recommendation/application"
)

type Handlers struct {
	getSuggestions *application.GetSuggestionsUseCase
	generate       *application.GenerateSuggestionsUseCase
}

func NewHandlers(getSuggestions *application.GetSuggestionsUseCase, generate *application.GenerateSuggestionsUseCase) *Handlers {
	return &Handlers{getSuggestions: getSuggestions, generate: generate}
}

// RegisterRoutes is mounted on its own top-level group in
// server.go (router.Group("/recommendations")), authenticated —
// suggestions are always for the logged-in caller, never looked up
// by id for someone else.
func (h *Handlers) RegisterRoutes(r *gin.RouterGroup) {
	r.GET("", h.GetSuggestions)
	r.POST("/refresh", h.RefreshSuggestions)
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

// GetSuggestions handles GET /recommendations?limit=20. limit is
// optional; an invalid or missing value falls back to
// application.DefaultLimit inside the use case, so this handler
// doesn't need its own default.
func (h *Handlers) GetSuggestions(c *gin.Context) {
	userID, err := accountIDFromContext(c)
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

	scores, err := h.getSuggestions.Execute(c.Request.Context(), userID, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal error"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"suggestions": scores})
}

// RefreshSuggestions handles POST /recommendations/refresh. It
// forces a live recompute for the logged-in caller right now,
// bypassing the cache entirely — useful for testing (proving a new
// signal moved after an action) and as a manual "refresh my
// suggestions" tool, without waiting for the background worker's
// next pass (Step 5).
func (h *Handlers) RefreshSuggestions(c *gin.Context) {
	userID, err := accountIDFromContext(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	scores, err := h.generate.Execute(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal error"})
		return
	}

	suggestions, err := h.getSuggestions.EnrichWithActivities(c.Request.Context(), scores)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal error"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"suggestions": suggestions})
}
