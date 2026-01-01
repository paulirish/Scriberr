package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"strings"

	"scriberr/internal/config"
	"scriberr/internal/database"
	"scriberr/internal/models"
	"scriberr/internal/repository"
	"scriberr/internal/transcription/interfaces"
)

func main() {
	// 1. Load Config and Database
	cfg := config.Load()
	if err := database.Initialize(cfg.DatabasePath); err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	defer database.Close()

	verbose := flag.Bool("v", false, "Show reasons for why a job is marked as unidentified")
	flag.Parse()

	ctx := context.Background()
	jobRepo := repository.NewJobRepository(database.DB)

	fmt.Println("Searching for jobs missing TitaNet speaker identification...")

	// 2. Fetch all completed jobs
	jobs, _, err := jobRepo.List(ctx, 0, -1)
	if err != nil {
		log.Fatalf("Failed to list jobs: %v", err)
	}

	unidentifiedCount := 0
	totalCompleted := 0

	for _, job := range jobs {
		if job.Status != models.StatusCompleted {
			continue
		}
		totalCompleted++

		isUnidentified := false
		reason := ""

		// Check 1: Does it have entries in speaker_job_centroids?
		var centroidCount int64
		database.DB.Model(&models.SpeakerJobCentroid{}).Where("transcription_job_id = ?", job.ID).Count(&centroidCount)
		
		if centroidCount == 0 {
			isUnidentified = true
			reason = "No entries in speaker_job_centroids table"
		} else if job.Transcript != nil && *job.Transcript != "" {
			// Check 2: Does the transcript actually contain 'global:' prefixes?
			var transcript interfaces.TranscriptResult
			if err := json.Unmarshal([]byte(*job.Transcript), &transcript); err == nil {
				hasGlobal := false
				for _, seg := range transcript.Segments {
					if seg.Speaker != nil && strings.HasPrefix(*seg.Speaker, "global:") {
						hasGlobal = true
						break
					}
				}
				if !hasGlobal {
					isUnidentified = true
					reason = "Transcript exists but contains no 'global:' speaker IDs"
				}
			}
		}

		if isUnidentified {
			unidentifiedCount++
			title := "Untitled"
			if job.Title != nil {
				title = *job.Title
			}
			fmt.Printf("[%s] %-40s", job.ID, title)
			if *verbose {
				fmt.Printf(" (Reason: %s)", reason)
			}
			fmt.Println()
		}
	}

	fmt.Printf("\nSummary:\n")
	fmt.Printf("Total Completed Jobs: %d\n", totalCompleted)
	fmt.Printf("Jobs Missing TitaNet Data: %d\n", unidentifiedCount)
}
