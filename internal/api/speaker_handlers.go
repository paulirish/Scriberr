package api

import (
	"net/http"
	"os"
	"os/exec"
	"strconv"

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

	segments, err := h.jobRepo.GetSegmentsBySpeakerID(c.Request.Context(), id)
	if err != nil {
		logger.Error("Failed to get speaker segments", "error", err)
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: err.Error()})
		return
	}

	c.JSON(http.StatusOK, segments)
}

// GetSpeakerSegmentAudio returns the audio for a specific speaker segment
// @Summary Get speaker segment audio
// @Description Get the audio for a specific speaker segment, sliced from the original file
// @Tags speakers
// @Produce audio/mpeg
// @Param id path string true "Speaker ID"
// @Param segment_id path int true "Segment ID"
// @Success 200 {file} binary
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/v1/speakers/{id}/segments/{segment_id}/audio [get]
func (h *Handler) GetSpeakerSegmentAudio(c *gin.Context) {
	segmentIDStr := c.Param("segment_id")
	segmentID, err := strconv.ParseUint(segmentIDStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "Invalid segment ID"})
		return
	}

	segment, err := h.jobRepo.GetSpeakerSegmentByID(c.Request.Context(), uint(segmentID))
	if err != nil {
		c.JSON(http.StatusNotFound, ErrorResponse{Error: "Segment not found"})
		return
	}

	job := segment.TranscriptionJob
	audioPath := job.AudioPath
	if job.IsMultiTrack && job.MergedAudioPath != nil && *job.MergedAudioPath != "" {
		if _, err := os.Stat(*job.MergedAudioPath); err == nil {
			audioPath = *job.MergedAudioPath
		}
	}

	if _, err := os.Stat(audioPath); os.IsNotExist(err) {
		c.JSON(http.StatusNotFound, ErrorResponse{Error: "Audio file not found"})
		return
	}

	// Use ffmpeg to slice the audio segment
	// ffmpeg -ss [start] -t [duration] -i [input] -f mp3 -
	duration := segment.End - segment.Start
	if duration <= 0 {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "Invalid segment duration"})
		return
	}

	// Set content type
	c.Header("Content-Type", "audio/mpeg")
	c.Header("Transfer-Encoding", "chunked")

	cmd := exec.Command("ffmpeg",
		"-ss", strconv.FormatFloat(segment.Start, 'f', 3, 64),
		"-t", strconv.FormatFloat(duration, 'f', 3, 64),
		"-i", audioPath,
		"-f", "mp3",
		"-acodec", "libmp3lame",
		"-ab", "128k",
		"pipe:1",
	)

	cmd.Stdout = c.Writer
	cmd.Stderr = os.Stderr // Log errors to stderr

	if err := cmd.Run(); err != nil {
		logger.Error("Failed to slice audio", "error", err)
		// We can't send a JSON error here because headers are already sent
		return
	}
}
