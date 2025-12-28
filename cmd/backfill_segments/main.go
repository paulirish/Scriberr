package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"scriberr/internal/config"
	"scriberr/internal/database"
	"scriberr/internal/models"
	"scriberr/internal/repository"
	"scriberr/internal/transcription/interfaces"
	"scriberr/pkg/logger"
)

func main() {
	logger.Init("info")
	cfg := config.Load()

	if err := database.Initialize(cfg.DatabasePath); err != nil {
		fmt.Printf("Failed to connect to database: %v\n", err)
		os.Exit(1)
	}
	defer database.Close()

	jobRepo := repository.NewJobRepository(database.DB)
	ctx := context.Background()

	// Get all completed jobs
	jobs, err := jobRepo.FindByStatus(ctx, models.StatusCompleted)
	if err != nil {
		fmt.Printf("Failed to fetch jobs: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Found %d completed jobs to process\n", len(jobs))

	totalSegments := 0
	for _, job := range jobs {
		if job.Transcript == nil || *job.Transcript == "" {
			continue
		}

		var result interfaces.TranscriptResult
		if err := json.Unmarshal([]byte(*job.Transcript), &result); err != nil {
			fmt.Printf("Failed to unmarshal transcript for job %s: %v\n", job.ID, err)
			continue
		}

		if len(result.Segments) == 0 {
			continue
		}

		fmt.Printf("Processing job %s (%d segments)...\n", job.ID, len(result.Segments))

		speakerSegments := make([]models.SpeakerSegment, 0, len(result.Segments))
		for _, seg := range result.Segments {
			speakerID := "Unknown"
			if seg.Speaker != nil {
				speakerID = *seg.Speaker
			}
			speakerSegments = append(speakerSegments, models.SpeakerSegment{
				TranscriptionJobID: job.ID,
				SpeakerID:          speakerID,
				Start:              seg.Start,
				End:                seg.End,
				Text:               seg.Text,
			})
		}

		if err := jobRepo.SaveSpeakerSegments(ctx, speakerSegments); err != nil {
			fmt.Printf("Failed to save speaker segments for job %s: %v\n", job.ID, err)
		} else {
			totalSegments += len(speakerSegments)
		}
	}

	fmt.Printf("Done! Created %d speaker segments.\n", totalSegments)
}
