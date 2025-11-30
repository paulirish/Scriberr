package api

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"scriberr/internal/transcription/adapters"
	"scriberr/pkg/logger"
)

// ListSpeakers returns all known speakers from the vector DB
// @Summary List speakers
// @Description Get a list of all identified speakers
// @Tags speakers
// @Produce json
// @Success 200 {array} adapters.SpeakerInfo
// @Failure 500 {object} ErrorResponse
// @Router /api/v1/speakers [get]
func (h *Handler) ListSpeakers(c *gin.Context) {
	// Initialize adapter (using the shared path or getting from a service registry would be better)
	// For now, we create a temporary instance or reusing logic would be ideal.
	// In a real dependency injection setup, the adapter should be part of the Handler struct or accessible via a Service.
	// Given the current architecture in handler.go, we might not have direct access to the specific UnifiedTranscriptionService instance.
	// However, we can re-instantiate the adapter wrapper as it's lightweight (just path config).

	// FIX: Ideally, the unified service or titanet adapter should be injected into Handler.
	// For this task, I will create a new instance pointing to the standard path.
	adapter := adapters.NewTitanetAdapter("data/models/nvidia/env")

	speakers, err := adapter.ListSpeakers(c.Request.Context())
	if err != nil {
		logger.Error("Failed to list speakers", "error", err)
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "Failed to list speakers"})
		return
	}

	c.JSON(http.StatusOK, speakers)
}

// RenameSpeakerRequest represents the request body for renaming
type RenameSpeakerRequest struct {
	Name string `json:"name" binding:"required"`
}

// RenameSpeaker updates a speaker's name
// @Summary Rename speaker
// @Description Rename an identified speaker
// @Tags speakers
// @Accept json
// @Produce json
// @Param id path string true "Speaker ID"
// @Param request body RenameSpeakerRequest true "New Name"
// @Success 200 {object} map[string]string
// @Failure 400 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/v1/speakers/{id} [put]
func (h *Handler) RenameSpeaker(c *gin.Context) {
	id := c.Param("id")
	var req RenameSpeakerRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "Invalid request body"})
		return
	}

	adapter := adapters.NewTitanetAdapter("data/models/nvidia/env")

	if err := adapter.RenameSpeaker(c.Request.Context(), id, req.Name); err != nil {
		logger.Error("Failed to rename speaker", "error", err)
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "Failed to rename speaker"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "success"})
}

// DeleteSpeaker deletes a speaker profile
// @Summary Delete speaker
// @Description Delete a speaker identity
// @Tags speakers
// @Produce json
// @Param id path string true "Speaker ID"
// @Success 200 {object} map[string]string
// @Failure 500 {object} ErrorResponse
// @Router /api/v1/speakers/{id} [delete]
func (h *Handler) DeleteSpeaker(c *gin.Context) {
	id := c.Param("id")

	adapter := adapters.NewTitanetAdapter("data/models/nvidia/env")

	if err := adapter.DeleteSpeaker(c.Request.Context(), id); err != nil {
		logger.Error("Failed to delete speaker", "error", err)
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "Failed to delete speaker"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "success"})
}
