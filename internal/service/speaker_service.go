package service

import (
	"context"
	"fmt"
	"strings"

	"scriberr/internal/repository"
	"scriberr/internal/transcription/interfaces"
	"scriberr/internal/transcription/registry"
	"scriberr/pkg/logger"
)

// SpeakerService handles business logic related to speakers.
type SpeakerService struct {
	jobRepo     repository.JobRepository
	speakerRepo repository.SpeakerRepository
	registry    *registry.ModelRegistry
}

// NewSpeakerService creates a new SpeakerService.
func NewSpeakerService(jobRepo repository.JobRepository, speakerRepo repository.SpeakerRepository) *SpeakerService {
	return &SpeakerService{
		jobRepo:     jobRepo,
		speakerRepo: speakerRepo,
		registry:    registry.GetRegistry(),
	}
}

// getAdapter retrieves the speaker management adapter from the registry.
func (s *SpeakerService) getAdapter() (interfaces.SpeakerManagementAdapter, error) {
	adapter, err := s.registry.GetIdentificationAdapter("titanet")
	if err != nil {
		return nil, fmt.Errorf("failed to get speaker identification adapter: %w", err)
	}

	mgmtAdapter, ok := adapter.(interfaces.SpeakerManagementAdapter)
	if !ok {
		return nil, fmt.Errorf("adapter does not support management operations")
	}

	return mgmtAdapter, nil
}

// RenameSpeaker updates a speaker's name globally and in the SQLite registry.
func (s *SpeakerService) RenameSpeaker(ctx context.Context, speakerID, newName string) error {
	adapter, err := s.getAdapter()
	if err != nil {
		return err
	}

	// Ensure the ID is clean (remove global: prefix if present for the adapter call)
	pureID := strings.TrimPrefix(speakerID, "global:")

	// Step 1: Rename the speaker in the central identity store (Qdrant)
	logger.Info("Renaming speaker in global store", "speakerID", pureID, "newName", newName)
	if err := adapter.RenameSpeaker(ctx, pureID, newName); err != nil {
		return fmt.Errorf("failed to rename speaker in titanet adapter: %w", err)
	}

	// Step 2: Update the SQLite registry (Source of Truth for resolution)
	// We use the prefixed ID for the SQLite table to match resolution logic
	fullID := "global:" + pureID
	if err := s.speakerRepo.UpdateName(ctx, fullID, newName); err != nil {
		logger.Error("Failed to update speaker name in SQLite registry", "id", fullID, "error", err)
		// We don't fail the whole operation if the SQLite cache update fails,
		// but it might cause a stale name until the next identification run.
	}

	logger.Info("Speaker renamed successfully", "speakerID", fullID, "newName", newName)
	return nil
}

// GetSpeaker retrieves a single speaker from the vector DB.
func (s *SpeakerService) GetSpeaker(ctx context.Context, speakerID string) (*interfaces.SpeakerInfo, error) {
	adapter, err := s.getAdapter()
	if err != nil {
		return nil, err
	}
	return adapter.GetSpeaker(ctx, speakerID)
}

func (s *SpeakerService) ListSpeakers(ctx context.Context) ([]interfaces.SpeakerInfo, error) {
	adapter, err := s.getAdapter()
	if err != nil {
		return nil, err
	}
	return adapter.ListSpeakers(ctx)
}

func (s *SpeakerService) DeleteSpeaker(ctx context.Context, speakerID string) error {
	adapter, err := s.getAdapter()
	if err != nil {
		return err
	}
	// Note: Deleting a speaker could also have a retroactive effect (e.g., anonymizing them in old transcripts).
	// For now, we just delete them from the global store.
	return adapter.DeleteSpeaker(ctx, speakerID)
}



