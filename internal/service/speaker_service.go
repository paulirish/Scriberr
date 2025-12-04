package service

import (
	"context"
	"encoding/json"
	"fmt"
	"scriberr/internal/repository"
	"scriberr/internal/transcription/adapters"
	"scriberr/pkg/logger"
)

// SpeakerService handles business logic related to speakers.
type SpeakerService struct {
	jobRepo        repository.JobRepository
	titanetAdapter *adapters.TitanetAdapter
}

// NewSpeakerService creates a new SpeakerService.
func NewSpeakerService(jobRepo repository.JobRepository) *SpeakerService {
	// The adapter is instantiated here to be used by the service.
	// The path might need to be configurable in the future.
	adapter := adapters.NewTitanetAdapter("data/whisperx-env/parakeet/")
	return &SpeakerService{
		jobRepo:        jobRepo,
		titanetAdapter: adapter,
	}
}

// RenameSpeaker updates a speaker's name globally and retroactively in all transcripts.
func (s *SpeakerService) RenameSpeaker(ctx context.Context, speakerID, newName string) error {
	// Step 1: Get the speaker's current name before renaming.
	logger.Info("Fetching current speaker name", "speakerID", speakerID)
	speakerInfo, err := s.titanetAdapter.GetSpeaker(ctx, speakerID)
	if err != nil {
		return fmt.Errorf("failed to get speaker info for ID %s: %w", speakerID, err)
	}
	oldName := speakerInfo.Name

	// If the name is already the new name, there's nothing to do.
	if oldName == newName {
		logger.Info("Speaker name is already up to date", "speakerID", speakerID, "name", newName)
		return nil
	}

	// Step 2: Rename the speaker in the central identity store (Qdrant)
	logger.Info("Renaming speaker in global store", "speakerID", speakerID, "oldName", oldName, "newName", newName)
	if err := s.titanetAdapter.RenameSpeaker(ctx, speakerID, newName); err != nil {
		return fmt.Errorf("failed to rename speaker in titanet adapter: %w", err)
	}

	// Step 3: Find all jobs and update their transcripts retroactively.
	logger.Info("Starting retroactive update of transcripts for speaker", "speakerID", speakerID)
	jobs, _, err := s.jobRepo.List(ctx, 0, -1)
	if err != nil {
		return fmt.Errorf("failed to get all jobs: %w", err)
	}

	updateCount := 0
	for _, job := range jobs {
		// Only process completed jobs with a transcript.
		if job.Status != "completed" || job.Transcript == nil || *job.Transcript == "" {
			continue
		}

		// Use a generic map to avoid losing fields, instead of a struct
		var transcriptData map[string]interface{}
		if err := json.Unmarshal([]byte(*job.Transcript), &transcriptData); err != nil {
			logger.Warn("Failed to unmarshal transcript for job", "jobID", job.ID, "error", err)
			continue
		}

		segments, ok := transcriptData["segments"].([]interface{})
		if !ok {
			continue
		}

		needsUpdate := false
		for _, segmentInterface := range segments {
			segment, ok := segmentInterface.(map[string]interface{})
			if !ok {
				continue
			}

			if speakerName, ok := segment["speaker"].(string); ok && speakerName == oldName {
				segment["speaker"] = newName
				needsUpdate = true
			}
		}

		if needsUpdate {
			newTranscriptBytes, err := json.Marshal(transcriptData)
			if err != nil {
				logger.Warn("Failed to marshal updated transcript for job", "jobID", job.ID, "error", err)
				continue
			}
			transcriptStr := string(newTranscriptBytes)
			job.Transcript = &transcriptStr
			if err := s.jobRepo.Update(ctx, &job); err != nil {
				logger.Warn("Failed to update job with new transcript", "jobID", job.ID, "error", err)
				continue
			}
			updateCount++
		}
	}

	logger.Info("Retroactive transcript update completed", "speakerID", speakerID, "updated_transcripts", updateCount)

	return nil
}

// GetSpeaker retrieves a single speaker from the vector DB.
func (s *SpeakerService) GetSpeaker(ctx context.Context, speakerID string) (*adapters.SpeakerInfo, error) {
	return s.titanetAdapter.GetSpeaker(ctx, speakerID)
}

// TODO: Add a method to get a speaker by ID from Qdrant to get the old name before renaming.
// This would involve adding a new command to the `titanet_manage.py` script and a corresponding
// method in the `TitanetAdapter`.

func (s *SpeakerService) ListSpeakers(ctx context.Context) ([]adapters.SpeakerInfo, error) {
	return s.titanetAdapter.ListSpeakers(ctx)
}

func (s *SpeakerService) DeleteSpeaker(ctx context.Context, speakerID string) error {
	// Note: Deleting a speaker could also have a retroactive effect (e.g., anonymizing them in old transcripts).
	// For now, we just delete them from the global store.
	return s.titanetAdapter.DeleteSpeaker(ctx, speakerID)
}



