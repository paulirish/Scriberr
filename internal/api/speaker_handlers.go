package api

import (
	"net/http"

	"scriberr/pkg/logger"

	"github.com/gin-gonic/gin"
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
	speakers, err := h.speakerService.ListSpeakers(c.Request.Context())
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
// @Description Rename an identified speaker and update past transcripts
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

	if err := h.speakerService.RenameSpeaker(c.Request.Context(), id, req.Name); err != nil {
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

	if err := h.speakerService.DeleteSpeaker(c.Request.Context(), id); err != nil {
		logger.Error("Failed to delete speaker", "error", err)
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "Failed to delete speaker"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "success"})
}
