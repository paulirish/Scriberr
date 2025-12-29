package api

import (
	"net/http"
	"strings"

	"scriberr/pkg/logger"

	"github.com/gin-gonic/gin"
)

// ListSpeakers returns all known speakers from the vector DB
// @Summary List speakers
// @Description Get a list of all identified speakers
// @Tags speakers
// @Produce json
// @Success 200 {array} interface{}
// @Failure 500 {object} ErrorResponse
// @Router /api/v1/speakers [get]
func (h *Handler) ListSpeakers(c *gin.Context) {
	speakers, err := h.speakerService.ListSpeakers(c.Request.Context())
	if err != nil {
		logger.Error("Failed to list speakers", "error", err)
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: err.Error()})
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
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: err.Error()})
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
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "success"})
}

// GetSpeakerSegments returns all audio segments associated with a speaker
// @Summary Get speaker segments
// @Description Get all audio segments and their associated transcription jobs for a speaker
// @Tags speakers
// @Produce json
// @Param id path string true "Speaker ID"
// @Success 200 {array} models.SpeakerSegment
// @Failure 500 {object} ErrorResponse
// @Router /api/v1/speakers/{id}/segments [get]
func (h *Handler) GetSpeakerSegments(c *gin.Context) {
	id := c.Param("id")

	// Try multiple possible ID formats that might be in the SQLite database
	ids := []string{id}
	// TitaNet identify script uses various prefixes with either full or short IDs
	if len(id) >= 8 {
		shortID := id[:8]
		// Add "Speaker-shortID" (Found in current database)
		ids = append(ids, "Speaker-"+shortID)
		// Add "Spk-shortID" (Used in some enrollment versions)
		ids = append(ids, "Spk-"+shortID)
	}

	// Also add the full ID with prefixes just in case
	if !strings.HasPrefix(id, "Speaker-") {
		ids = append(ids, "Speaker-"+id)
	}
	if !strings.HasPrefix(id, "Spk-") {
		ids = append(ids, "Spk-"+id)
	}

	segments, err := h.jobRepo.GetSegmentsBySpeakerIDs(c.Request.Context(), ids)
	if err != nil {
		logger.Error("Failed to get speaker segments", "error", err)
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: err.Error()})
		return
	}

	c.JSON(http.StatusOK, segments)
}
