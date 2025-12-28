package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"sort"

	"scriberr/internal/config"
	"scriberr/internal/database"
	"scriberr/internal/models"
	"scriberr/internal/repository"
	"scriberr/internal/transcription/interfaces"
	"scriberr/pkg/logger"
)

func main() {
	clearFlag := flag.Bool("clear", false, "Clear existing speaker segments before backfilling")
	flag.Parse()

	logger.Init("info")
	cfg := config.Load()

	if err := database.Initialize(cfg.DatabasePath); err != nil {
		fmt.Printf("Failed to connect to database: %v\n", err)
		os.Exit(1)
	}
	defer database.Close()

	// Clear existing segments if requested
	if *clearFlag {
		fmt.Println("Clearing existing speaker segments and centroids...")
		database.DB.Exec("DELETE FROM speaker_segments")
		database.DB.Exec("DELETE FROM speaker_job_centroids")
	}

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

		// Group segments by speaker
		speakerSegmentsMap := make(map[string][]interfaces.TranscriptSegment)
		for _, seg := range result.Segments {
			speakerID := "Unknown"
			if seg.Speaker != nil {
				speakerID = *seg.Speaker
			}
			speakerSegmentsMap[speakerID] = append(speakerSegmentsMap[speakerID], seg)
		}

		fmt.Printf("Processing job %s (%d speakers)...\n", job.ID, len(speakerSegmentsMap))

		for speakerID, segments := range speakerSegmentsMap {
			// Sort segments by duration (descending)
			sort.Slice(segments, func(i, j int) bool {
				durI := segments[i].End - segments[i].Start
				durJ := segments[j].End - segments[j].Start
				return durI > durJ
			})

			// Take top 10 (matching TitaNet identification logic)
			limit := 10
			if len(segments) < limit {
				limit = len(segments)
			}
			topSegments := segments[:limit]

			dbSegments := make([]models.SpeakerSegment, 0, len(topSegments))
			for _, seg := range topSegments {
				dbSegments = append(dbSegments, models.SpeakerSegment{
					TranscriptionJobID: job.ID,
					SpeakerID:          speakerID,
					Start:              seg.Start,
					End:                seg.End,
					Text:               seg.Text,
				})
			}

			if err := jobRepo.SaveSpeakerSegments(ctx, dbSegments); err != nil {
				fmt.Printf("Failed to save speaker segments for speaker %s in job %s: %v\n", speakerID, job.ID, err)
			} else {
				totalSegments += len(dbSegments)
			}
		}
	}

	fmt.Printf("Done! Created %d reference speaker segments.\n", totalSegments)
}