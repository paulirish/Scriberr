package main

import (
	"context"
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"flag"
	"log"
	"regexp"
	"strings"

	"scriberr/internal/config"
	"scriberr/internal/database"
	"scriberr/internal/models"
	"scriberr/internal/repository"
	"scriberr/internal/transcription/interfaces"
)

func main() {
	// 0. Parse Flags
	dryRun := flag.Bool("dry-run", false, "Preview changes without modifying the database")
	flag.Parse()

	// 1. Load Config and Database
	cfg := config.Load()

	if err := database.Initialize(cfg.DatabasePath); err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	defer database.Close()

	if *dryRun {
		log.Println("=== DRY RUN MODE: No changes will be committed ===")
	}

	ctx := context.Background()
	jobRepo := repository.NewJobRepository(database.DB)
	speakerRepo := repository.NewSpeakerRepository(database.DB)

	log.Println("Starting speaker migration...")

	// 2. Fetch all completed jobs
	jobs, _, err := jobRepo.List(ctx, 0, -1)
	if err != nil {
		log.Fatalf("Failed to list jobs: %v", err)
	}

	speakerRegex := regexp.MustCompile(`^speaker_\d+$`)
	globalNameCache := make(map[string]string) // Name -> Global ID

	updateCount := 0
	for _, job := range jobs {
		if job.Status != models.StatusCompleted || job.Transcript == nil || *job.Transcript == "" {
			continue
		}

		log.Printf("Processing Job: %s (%s)", job.ID, *job.Title)

		var transcript interfaces.TranscriptResult
		if err := json.Unmarshal([]byte(*job.Transcript), &transcript); err != nil {
			log.Printf("  [WARN] Failed to unmarshal transcript for job %s: %v", job.ID, err)
			continue
		}

		jobNeedsUpdate := false
		localMapping := make(map[string]string) // Old Name -> New ID

		// Identify unique speakers in this job
		for i := range transcript.Segments {
			seg := &transcript.Segments[i]
			if seg.Speaker == nil || *seg.Speaker == "" {
				continue
			}

			oldName := *seg.Speaker
			if strings.HasPrefix(oldName, "local:") || strings.HasPrefix(oldName, "global:") {
				continue // Already migrated
			}

			newID, exists := localMapping[oldName]
			if !exists {
				if speakerRegex.MatchString(oldName) {
					// It's a local speaker (e.g. speaker_00)
					newID = "local:" + oldName
				} else {
					// It's a global name (e.g. John Doe)
					globalID, cached := globalNameCache[oldName]
					if !cached {
						// Create a stable legacy ID based on name hash
						hash := sha1.Sum([]byte(oldName))
						globalID = "global:legacy-" + hex.EncodeToString(hash[:8])
						
						// Seed the speakers table
						speaker := &models.Speaker{
							ID:   globalID,
							Name: oldName,
						}
						if !*dryRun {
							if err := speakerRepo.Update(ctx, speaker); err != nil {
								log.Printf("  [ERROR] Failed to seed speaker %s: %v", oldName, err)
							}
						}
						globalNameCache[oldName] = globalID
						log.Printf("  [INFO] Global identity: %s -> %s", oldName, globalID)
					}
					newID = globalID
				}
				localMapping[oldName] = newID
			}

			// Update Segment
			seg.Speaker = &newID
			jobNeedsUpdate = true
		}

		// Update Words
		for i := range transcript.WordSegments {
			word := &transcript.WordSegments[i]
			if word.Speaker != nil {
				if newID, ok := localMapping[*word.Speaker]; ok {
					word.Speaker = &newID
					jobNeedsUpdate = true
				}
			}
		}

		if jobNeedsUpdate {
			if *dryRun {
				log.Printf("  [DRY RUN] Would update job %s and related tables for %d speakers", job.ID, len(localMapping))
				updateCount++
				continue
			}

			// Save updated transcript
			newJSON, _ := json.Marshal(transcript)
			transcriptStr := string(newJSON)
			job.Transcript = &transcriptStr

			// Update IndividualTranscripts if present (Multi-track)
			if job.IndividualTranscripts != nil && *job.IndividualTranscripts != "" {
				var individualTranscripts map[string]string
				if err := json.Unmarshal([]byte(*job.IndividualTranscripts), &individualTranscripts); err == nil {
					updatedIndividual := false
					for trackName, tJSON := range individualTranscripts {
						var tResult interfaces.TranscriptResult
						if err := json.Unmarshal([]byte(tJSON), &tResult); err == nil {
							tNeedsUpdate := false
							for i := range tResult.Segments {
								seg := &tResult.Segments[i]
								if seg.Speaker != nil {
									if newID, ok := localMapping[*seg.Speaker]; ok {
										seg.Speaker = &newID
										tNeedsUpdate = true
									}
								}
							}
							if tNeedsUpdate {
								newTJSON, _ := json.Marshal(tResult)
								individualTranscripts[trackName] = string(newTJSON)
								updatedIndividual = true
							}
						}
					}
					if updatedIndividual {
						newIndividualJSON, _ := json.Marshal(individualTranscripts)
						individualJSONStr := string(newIndividualJSON)
						job.IndividualTranscripts = &individualJSONStr
					}
				}
			}

			if err := jobRepo.Update(ctx, &job); err != nil {
				log.Printf("  [ERROR] Failed to update job %s: %v", job.ID, err)
				continue
			}

			// Update speaker_segments table
			for oldName, newID := range localMapping {
				if err := database.DB.Model(&models.SpeakerSegment{}).
					Where("transcription_job_id = ? AND speaker_id = ?", job.ID, oldName).
					Update("speaker_id", newID).Error; err != nil {
					log.Printf("  [WARN] Failed to update speaker_segments for job %s: %v", job.ID, err)
				}
				
				// Update speaker_mappings table (the original_speaker column)
				if err := database.DB.Model(&models.SpeakerMapping{}).
					Where("transcription_job_id = ? AND original_speaker = ?", job.ID, oldName).
					Update("original_speaker", newID).Error; err != nil {
					log.Printf("  [WARN] Failed to update speaker_mappings for job %s: %v", job.ID, err)
				}

				// Update speaker_job_centroids table
				if err := database.DB.Model(&models.SpeakerJobCentroid{}).
					Where("transcription_job_id = ? AND speaker_id = ?", job.ID, oldName).
					Update("speaker_id", newID).Error; err != nil {
					log.Printf("  [WARN] Failed to update speaker_job_centroids for job %s: %v", job.ID, err)
				}
			}

			updateCount++
			log.Printf("  [SUCCESS] Migrated %d speakers in job %s", len(localMapping), job.ID)
		}
	}

	log.Printf("Migration complete! Updated %d jobs.", updateCount)
}
