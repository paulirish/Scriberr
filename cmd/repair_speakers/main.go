package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"scriberr/internal/config"
	"scriberr/internal/database"
	"scriberr/internal/models"
	"scriberr/internal/repository"
	"scriberr/internal/transcription/interfaces"
	"scriberr/internal/transcription/pipeline"
	"scriberr/internal/transcription/registry"
	"scriberr/pkg/logger"
)

type ffprobeOutput struct {
	Streams []struct {
		CodecType  string `json:"codec_type"`
		SampleRate string `json:"sample_rate"`
		Channels   int    `json:"channels"`
		Duration   string `json:"duration"`
		CodecName  string `json:"codec_name"`
		BitRate    string `json:"bit_rate"`
	} `json:"streams"`
	Format struct {
		Duration string `json:"duration"`
		Size     string `json:"size"`
	} `json:"format"`
}

func createAudioInput(audioPath string) (interfaces.AudioInput, error) {
	fileInfo, err := os.Stat(audioPath)
	if err != nil {
		return interfaces.AudioInput{}, fmt.Errorf("failed to stat audio file: %w", err)
	}

	ext := strings.ToLower(filepath.Ext(audioPath))
	format := strings.TrimPrefix(ext, ".")

	audioInput := interfaces.AudioInput{
		FilePath: audioPath,
		Format:   format,
		Size:     fileInfo.Size(),
		Metadata: map[string]string{},
	}

	cmd := exec.Command("ffprobe",
		"-v", "quiet",
		"-print_format", "json",
		"-show_format",
		"-show_streams",
		audioPath)

	output, err := cmd.Output()
	if err != nil {
		audioInput.SampleRate = 16000
		audioInput.Channels = 1
		return audioInput, nil
	}

	var probeData ffprobeOutput
	if err := json.Unmarshal(output, &probeData); err != nil {
		audioInput.SampleRate = 16000
		audioInput.Channels = 1
		return audioInput, nil
	}

	for _, stream := range probeData.Streams {
		if stream.CodecType == "audio" {
			if sampleRate, err := strconv.Atoi(stream.SampleRate); err == nil {
				audioInput.SampleRate = sampleRate
			}
			audioInput.Channels = stream.Channels
			if duration, err := strconv.ParseFloat(stream.Duration, 64); err == nil {
				audioInput.Duration = time.Duration(duration * float64(time.Second))
			}
			break
		}
	}

	if audioInput.SampleRate == 0 {
		audioInput.SampleRate = 16000
	}
	if audioInput.Channels == 0 {
		audioInput.Channels = 1
	}

	return audioInput, nil
}

func main() {
	dryRun := flag.Bool("dry-run", false, "Don't actually update the database")
	jobID := flag.String("job", "", "Specific job ID to repair")
	force := flag.Bool("force", false, "Repair even if speaker_0 is not found")
	flag.Parse()

	logger.Init("info")
	cfg := config.Load()

	if err := database.Initialize(cfg.DatabasePath); err != nil {
		fmt.Printf("Failed to connect to database: %v\n", err)
		os.Exit(1)
	}
	defer database.Close()

	registry.RegisterStandardAdapters(cfg)
	
	ctx := context.Background()
	if err := registry.GetRegistry().InitializeModelsSync(ctx); err != nil {
		fmt.Printf("Failed to initialize models: %v\n", err)
		os.Exit(1)
	}

	jobRepo := repository.NewJobRepository(database.DB)
	audioPipeline := pipeline.NewProcessingPipeline()

	var jobs []models.TranscriptionJob
	if *jobID != "" {
		job, err := jobRepo.FindByID(ctx, *jobID)
		if err != nil {
			fmt.Printf("Failed to fetch job %s: %v\n", *jobID, err)
			os.Exit(1)
		}
		jobs = []models.TranscriptionJob{*job}
	} else {
		query := database.DB.Where("updated_at >= ?", "2026-01-20")
		if !*force {
			query = query.Where("transcript LIKE ?", "%speaker_0%")
		}
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

		diarizationResult := &interfaces.DiarizationResult{
			Segments: make([]interfaces.DiarizationSegment, len(result.Segments)),
		}
		for i, seg := range result.Segments {
			speaker := ""
			if seg.Speaker != nil {
				speaker = *seg.Speaker
			}
			diarizationResult.Segments[i] = interfaces.DiarizationSegment{
				Start:   seg.Start,
				End:     seg.End,
				Speaker: speaker,
			}
		}

		audioInput, err := createAudioInput(job.AudioPath)
		if err != nil {
			fmt.Printf("  Error probing audio: %v\n", err)
			continue
		}

		fmt.Printf("  Preprocessing audio...\n")
		processedInput, err := audioPipeline.ProcessAudio(ctx, audioInput, identificationAdapter.GetCapabilities())
		if err != nil {
			fmt.Printf("  Preprocessing failed: %v\n", err)
			continue
		}
		
		// Ensure cleanup of temporary file
		if processedInput.TempFilePath != "" {
			defer os.Remove(processedInput.TempFilePath)
		}

		procCtx := interfaces.ProcessingContext{
			JobID:           job.ID,
			OutputDirectory: filepath.Join(cfg.TempDir, "repair", job.ID),
			TempDirectory:   cfg.TempDir,
		}
		os.MkdirAll(procCtx.OutputDirectory, 0755)

		fmt.Printf("  Running speaker identification...\n")
		identifiedResult, err := identificationAdapter.IdentifySpeakers(ctx, processedInput, diarizationResult, nil, procCtx)
		if err != nil {
			fmt.Printf("  Identification failed: %v\n", err)
			os.RemoveAll(procCtx.OutputDirectory)
			continue
		}

		speakerMap := make(map[string]string)
		for i, seg := range identifiedResult.Segments {
			if i < len(result.Segments) {
				oldSpeaker := ""
				if result.Segments[i].Speaker != nil {
					oldSpeaker = *result.Segments[i].Speaker
				}
				
				newSpeaker := seg.Speaker
				result.Segments[i].Speaker = &newSpeaker
				
				if oldSpeaker != "" && oldSpeaker != newSpeaker {
					speakerMap[oldSpeaker] = newSpeaker
				}
			}
		}

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
			}

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
		
		os.RemoveAll(procCtx.OutputDirectory)
		if processedInput.TempFilePath != "" {
			os.Remove(processedInput.TempFilePath)
		}
	}

	fmt.Println("Repair process completed.")
}
