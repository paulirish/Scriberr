package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"scriberr/internal/config"
	"scriberr/internal/database"
	"scriberr/internal/models"
	"scriberr/internal/repository"
	"scriberr/internal/transcription/interfaces"
	"scriberr/internal/transcription/registry"
	"scriberr/pkg/logger"
)

func main() {
	dryRun := flag.Bool("dry-run", false, "Don't actually update the database")
	jobID := flag.String("job", "", "Specific job ID to repair")
	flag.Parse()

	logger.Init("info")
	cfg := config.Load()

	if err := database.Initialize(cfg.DatabasePath); err != nil {
		fmt.Printf("Failed to connect to database: %v\n", err)
		os.Exit(1)
	}
	defer database.Close()

	registry.RegisterStandardAdapters(cfg)
	
	// Initialize models synchronously
	ctx := context.Background()
	if err := registry.GetRegistry().InitializeModelsSync(ctx); err != nil {
		fmt.Printf("Failed to initialize models: %v\n", err)
		os.Exit(1)
	}

	jobRepo := repository.NewJobRepository(database.DB)

	var jobs []models.TranscriptionJob
	if *jobID != "" {
		job, err := jobRepo.FindByID(ctx, *jobID)
		if err != nil {
			fmt.Printf("Failed to fetch job %s: %v\n", *jobID, err)
			os.Exit(1)
		}
		jobs = []models.TranscriptionJob{*job}
	} else {
		// Find jobs updated today with generic speaker names
		query := database.DB.Where("updated_at >= ? AND transcript LIKE ?", "2026-01-20", "%speaker_0%")
		if err := query.Find(&jobs).Error; err != nil {
			fmt.Printf("Failed to fetch jobs: %v\n", err)
			os.Exit(1)
		}
	}

	fmt.Printf("Found %d jobs to repair\n", len(jobs))

	identificationAdapter, err := registry.GetRegistry().GetIdentificationAdapter("titanet")
	if err != nil {
		fmt.Printf("Failed to get titanet adapter: %v\n", err)
		os.Exit(1)
	}

	for _, job := range jobs {
		title := ""
		if job.Title != nil {
			title = *job.Title
		}
		fmt.Printf("Repairing job %s (%s)...\n", job.ID, title)

		if job.Transcript == nil || *job.Transcript == "" {
			fmt.Printf("  Skipping: No transcript\n")
			continue
		}

		var result interfaces.TranscriptResult
		if err := json.Unmarshal([]byte(*job.Transcript), &result); err != nil {
			fmt.Printf("  Error unmarshaling transcript: %v\n", err)
			continue
		}

		// Convert TranscriptResult to DiarizationResult for IdentifySpeakers
		diarizationResult := &interfaces.DiarizationResult{
			Segments: make([]interfaces.DiarizationSegment, len(result.Segments)),
		}
		for i, seg := range result.Segments {
			speaker := ""
			if seg.Speaker != nil {
				speaker = *seg.Speaker
			}
			diarizationResult.Segments[i] = interfaces.DiarizationSegment{
				Start:	seg.Start,
				End:	seg.End,
				Speaker:	speaker,
			}
		}

		// Create audio input
		audioInput := interfaces.AudioInput{
			FilePath: job.AudioPath,
		}

		// Processing Context (minimal)
		procCtx := interfaces.ProcessingContext{
			JobID:           job.ID,
			OutputDirectory: filepath.Join(cfg.TempDir, "repair", job.ID),
			TempDirectory:   cfg.TempDir,
		}
		os.MkdirAll(procCtx.OutputDirectory, 0755)

		fmt.Printf("  Running speaker identification...\n")
		identifiedResult, err := identificationAdapter.IdentifySpeakers(ctx, audioInput, diarizationResult, nil, procCtx)
		if err != nil {
			fmt.Printf("  Identification failed: %v\n", err)
			continue
		}

		// Merge back
		speakerMap := make(map[string]string) // local -> global
		for i, seg := range identifiedResult.Segments {
			if i < len(result.Segments) {
				oldSpeaker := ""
				if result.Segments[i].Speaker != nil {
					oldSpeaker = *result.Segments[i].Speaker
				}
				
				newSpeaker := seg.Speaker
				result.Segments[i].Speaker = &newSpeaker
				
				// Keep track of the mapping
				if oldSpeaker != "" && oldSpeaker != newSpeaker {
					speakerMap[oldSpeaker] = newSpeaker
				}
			}
		}

		// Also update word segments if they exist
		for i := range result.WordSegments {
			word := &result.WordSegments[i]
			if word.Speaker != nil {
				if global, ok := speakerMap[*word.Speaker]; ok {
					word.Speaker = &global
				}
			}
		}

		if !*dryRun {
			fmt.Printf("  Saving results to database...\n")
			
			updatedTranscript, _ := json.Marshal(result)
			transcriptStr := string(updatedTranscript)
			if err := jobRepo.UpdateTranscript(ctx, job.ID, transcriptStr); err != nil {
				fmt.Printf("  Failed to update transcript: %v\n", err)
				continue
			}

			// Save speaker segments
			speakerSegments := make([]models.SpeakerSegment, 0)
			for _, seg := range result.Segments {
				if !seg.IsReference {
					continue
				}
				
				spk := "Unknown"
				if seg.Speaker != nil {
					spk = *seg.Speaker
				}
				
				var embBytes []byte
				if len(seg.Embedding) > 0 {
					embBytes, _ = json.Marshal(seg.Embedding)
				}
				
				speakerSegments = append(speakerSegments, models.SpeakerSegment{
					TranscriptionJobID: job.ID,
					SpeakerID:          spk,
					Start:              seg.Start,
					End:                seg.End,
					Text:               seg.Text,
					Embedding:          embBytes,
				})
			}
			if err := jobRepo.SaveSpeakerSegments(ctx, speakerSegments); err != nil {
				fmt.Printf("  Failed to save speaker segments: %v\n", err)
			}

			// Save centroids
			if identifiedResult.SpeakerCentroids != nil {
				centroids := make([]models.SpeakerJobCentroid, 0)
				for spkID, centroid := range identifiedResult.SpeakerCentroids {
					cBytes, _ := json.Marshal(centroid)
					centroids = append(centroids, models.SpeakerJobCentroid{
						TranscriptionJobID: job.ID,
						SpeakerID:          spkID,
						Centroid:           cBytes,
					})
				}
				if err := jobRepo.SaveSpeakerJobCentroids(ctx, centroids); err != nil {
					fmt.Printf("  Failed to save centroids: %v\n", err)
				}
			}
			
			fmt.Printf("  Successfully repaired job %s\n", job.ID)
		} else {
			fmt.Printf("  Dry run: found mapping %v\n", speakerMap)
		}
		
		// Cleanup temp dir
		os.RemoveAll(procCtx.OutputDirectory)
	}

	fmt.Println("Repair process completed.")
}