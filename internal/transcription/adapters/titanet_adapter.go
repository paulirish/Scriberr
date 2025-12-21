package adapters

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"scriberr/internal/transcription/interfaces"
	"scriberr/pkg/downloader"
	"scriberr/pkg/logger"
)

// TitanetAdapter handles speaker identification using TitaNet and Qdrant
type TitanetAdapter struct {
	*BaseAdapter
	envPath string
}

// NewTitanetAdapter creates a new TitaNet adapter
func NewTitanetAdapter(envPath string) *TitanetAdapter {
	capabilities := interfaces.ModelCapabilities{
		ModelID:            "titanet",
		ModelFamily:        "nvidia_titanet",
		DisplayName:        "NVIDIA TitaNet Large",
		Description:        "Speaker identification and verification using TitaNet and Vector DB",
		Version:            "2.0.0",
		SupportedLanguages: []string{"*"},
		SupportedFormats:   []string{"wav", "flac"},
		RequiresGPU:        false,
		Features: map[string]bool{
			"speaker_identification": true,
			"persistent_identity":    true,
		},
	}

	// Schema for parameters
	schema := []interfaces.ParameterSchema{
		{
			Name:        "similarity_threshold",
			Type:        "float",
			Default:     0.7,
			Description: "Threshold for cosine similarity to identify a speaker.",
		},
		{
			Name:        "threshold_new",
			Type:        "float",
			Default:     0.55,
			Description: "Threshold below which a speaker is considered a new speaker.",
		},
		{
			Name:        "norm_threshold",
			Type:        "float",
			Default:     1.5,
			Description: "Z-score like threshold for matching when S-Norm is enabled.",
		},
		{
			Name:        "alpha_max",
			Type:        "float",
			Default:     0.25,
			Description: "Maximum learning rate for Confidence-Weighted EMA.",
		},
		{
			Name:        "min_duration_full_weight",
			Type:        "float",
			Default:     4.0,
			Description: "Minimum duration in seconds for a segment to receive full weight in EMA update.",
		},
	}

	baseAdapter := NewBaseAdapter("titanet", envPath, capabilities, schema)

	return &TitanetAdapter{
		BaseAdapter: baseAdapter,
		envPath:     envPath,
	}
}

// PrepareEnvironment ensures TitaNet model and dependencies are ready
func (t *TitanetAdapter) PrepareEnvironment(ctx context.Context) error {
	logger.Info("Preparing TitaNet environment", "env_path", t.envPath)

	if err := t.downloadTitanetModel(); err != nil {
		return fmt.Errorf("failed to download TitaNet model: %w", err)
	}

	// The python scripts are now part of the source code, so we don't need to create them dynamically.
	// We just need to ensure they are executable.
	scripts := []string{"titanet_identify_v2.py", "titanet_cohort_manager.py"}
	for _, script := range scripts {
		scriptPath := filepath.Join("data/whisperx-env/parakeet", script)
		if _, err := os.Stat(scriptPath); os.IsNotExist(err) {
			// This should not happen if the code is checked out correctly.
			return fmt.Errorf("required script not found: %s", scriptPath)
		}
		// In a real production environment, you might make it executable here.
		// For now, we assume it's handled by the environment/deployment.
	}

	t.initialized = true
	return nil
}

func (t *TitanetAdapter) downloadTitanetModel() error {
	modelFileName := "titanet-l.nemo"
	modelPath := filepath.Join(t.envPath, modelFileName)

	if stat, err := os.Stat(modelPath); err == nil && stat.Size() > 1024*1024 {
		return nil
	}

	logger.Info("Downloading TitaNet model", "path", modelPath)
	modelURL := "https://huggingface.co/nvidia/speakerverification_en_titanet_large/resolve/main/speakerverification_en_titanet_large.nemo?download=true"

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
	defer cancel()

	return downloader.DownloadFile(ctx, modelURL, modelPath)
}

// IdentifySpeakers runs the identification process
func (t *TitanetAdapter) IdentifySpeakers(ctx context.Context, input interfaces.AudioInput, diarizationResult *interfaces.DiarizationResult, params map[string]interface{}, procCtx interfaces.ProcessingContext) (*interfaces.DiarizationResult, error) {
	tempDir, err := t.CreateTempDirectory(procCtx)
	if err != nil {
		return nil, err
	}
	defer t.CleanupTempDirectory(tempDir)

	inputJSON := filepath.Join(tempDir, "input_segments.json")
	outputJSON := filepath.Join(tempDir, "output_segments.json")

	wrapper := map[string]interface{}{
		"segments": diarizationResult.Segments,
	}
	data, _ := json.Marshal(wrapper)
	if err := os.WriteFile(inputJSON, data, 0644); err != nil {
		return nil, fmt.Errorf("failed to write input json: %w", err)
	}

	scriptPath := filepath.Join("data/whisperx-env/parakeet", "titanet_identify_v2.py")
	qdrantHost := os.Getenv("QDRANT_HOST")
	if qdrantHost == "" {
		qdrantHost = "qdrant"
	}

	cmd := exec.CommandContext(ctx, "uv", "run", "--native-tls", "--project", t.envPath, "python", scriptPath,
		input.FilePath,
		inputJSON,
		outputJSON,
		"--qdrant", qdrantHost,
		"--threshold", fmt.Sprintf("%.2f", t.GetFloatParameter(params, "similarity_threshold")),
		"--threshold-new", fmt.Sprintf("%.2f", t.GetFloatParameter(params, "threshold_new")),
		"--norm-threshold", fmt.Sprintf("%.2f", t.GetFloatParameter(params, "norm_threshold")),
		"--alpha-max", fmt.Sprintf("%.2f", t.GetFloatParameter(params, "alpha_max")),
		"--min-duration-full-weight", fmt.Sprintf("%.2f", t.GetFloatParameter(params, "min_duration_full_weight")),
	)

	logger.Info("Executing Titanet command", "args", strings.Join(cmd.Args, " "))

	cmd.Env = append(os.Environ(), "PYTHONUNBUFFERED=1")
	output, err := cmd.CombinedOutput()
	if err != nil {
		logger.Error("Identity script failed", "output", string(output))
		return nil, fmt.Errorf("identity script failed: %w", err)
	}
	logger.Debug("Identity script output", "output", string(output))

	var speakerEvents []string
	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		if strings.Contains(line, " Matched ") || strings.Contains(line, " Enrolling ") {
			speakerEvents = append(speakerEvents, line)
		}
	}

	if len(speakerEvents) > 0 {
		logger.Info("Speaker identification events", "details", "\n"+strings.Join(speakerEvents, "\n"))
	}

	resultData, err := os.ReadFile(outputJSON)
	if err != nil {
		return nil, fmt.Errorf("failed to read identity output: %w", err)
	}

	var resultWrapper struct {
		Segments []interfaces.DiarizationSegment `json:"segments"`
	}
	if err := json.Unmarshal(resultData, &resultWrapper); err != nil {
		return nil, fmt.Errorf("failed to parse identity output: %w", err)
	}

	newResult := *diarizationResult
	newResult.Segments = resultWrapper.Segments

	speakerSet := make(map[string]bool)
	for _, seg := range newResult.Segments {
		speakerSet[seg.Speaker] = true
	}
	newResult.Speakers = make([]string, 0, len(speakerSet))
	for s := range speakerSet {
		newResult.Speakers = append(newResult.Speakers, s)
	}
	newResult.SpeakerCount = len(newResult.Speakers)

	return &newResult, nil
}

// ... (rest of the file is the same, with titanet_manage.py still embedded)
// ... I will skip modifying the management script for now to keep the change focused.

// SpeakerInfo represents a speaker in the vector DB
type SpeakerInfo struct {
	ID        string  `json:"id"`
	Name      string  `json:"name"`
	CreatedAt float64 `json:"created_at"`
}

// EnsureManagementScript creates the python script for managing speakers
func (t *TitanetAdapter) EnsureManagementScript() error {
	scriptPath := filepath.Join(t.envPath, "titanet_manage.py")
	if _, err := os.Stat(scriptPath); err == nil {
		return nil
	}

	content := `#!/usr/bin/env python3
"""
TitaNet Speaker Management Script
"""
import argparse
import json
import sys
import logging
from typing import List, Dict

# Setup logging
logging.basicConfig(level=logging.ERROR, format='%(message)s')
logger = logging.getLogger(__name__)

try:
    from qdrant_client import QdrantClient
    from qdrant_client.http import models as qmodels
except ImportError as e:
    logger.error(f"Import Error: {e}")
    sys.exit(1)

def setup_client(host: str):
    return QdrantClient(host=host, port=6333)

def list_speakers(host: str, collection: str):
    client = setup_client(host)
    try:
        points, _ = client.scroll(
            collection_name=collection,
            limit=1000,
            with_payload=True,
            with_vectors=False
        )

        speakers = []
        for p in points:
            payload = p.payload or {}
            speakers.append({
                "id": p.id,
                "name": payload.get("name", "Unknown"),
                "created_at": float(payload.get("created_at", 0))
            })

        print(json.dumps(speakers))
    except Exception as e:
        logger.error(f"Error listing speakers: {e}")
        sys.exit(1)

def get_speaker(host: str, collection: str, speaker_id: str):
    client = setup_client(host)
    try:
        points = client.retrieve(
            collection_name=collection,
            ids=[speaker_id],
            with_payload=True,
            with_vectors=False
        )
        if not points:
            logger.error("Speaker not found")
            sys.exit(1)
        
        p = points[0]
        payload = p.payload or {}
        speaker = {
            "id": p.id,
            "name": payload.get("name", "Unknown"),
            "created_at": float(payload.get("created_at", 0))
        }
        print(json.dumps(speaker))
    except Exception as e:
        logger.error(f"Error getting speaker: {e}")
        sys.exit(1)

def rename_speaker(host: str, collection: str, speaker_id: str, new_name: str):
    client = setup_client(host)
    try:
        points = client.retrieve(
            collection_name=collection,
            ids=[speaker_id]
        )

        if not points:
            logger.error("Speaker not found")
            sys.exit(1)

        client.set_payload(
            collection_name=collection,
            payload={"name": new_name},
            points=[speaker_id]
        )
        print(json.dumps({"status": "success", "id": speaker_id, "name": new_name}))
    except Exception as e:
        logger.error(f"Error renaming speaker: {e}")
        sys.exit(1)

def delete_speaker(host: str, collection: str, speaker_id: str):
    client = setup_client(host)
    try:
        client.delete(
            collection_name=collection,
            points_selector=qmodels.PointIdsList(points=[speaker_id])
        )
        print(json.dumps({"status": "success", "id": speaker_id}))
    except Exception as e:
        logger.error(f"Error deleting speaker: {e}")
        sys.exit(1)

if __name__ == "__main__":
    parser = argparse.ArgumentParser()
    subparsers = parser.add_subparsers(dest="command", required=True)

    parser_list = subparsers.add_parser("list")
    parser_list.add_argument("--qdrant", default="qdrant")
    parser_list.add_argument("--collection", default="speakers")

    parser_get = subparsers.add_parser("get")
    parser_get.add_argument("id")
    parser_get.add_argument("--qdrant", default="qdrant")
    parser_get.add_argument("--collection", default="speakers")

    parser_rename = subparsers.add_parser("rename")
    parser_rename.add_argument("id")
    parser_rename.add_argument("new_name")
    parser_rename.add_argument("--qdrant", default="qdrant")
    parser_rename.add_argument("--collection", default="speakers")

    parser_delete = subparsers.add_parser("delete")
    parser_delete.add_argument("id")
    parser_delete.add_argument("--qdrant", default="qdrant")
    parser_delete.add_argument("--collection", default="speakers")

    args = parser.parse_args()

    if args.command == "list":
        list_speakers(args.qdrant, args.collection)
    elif args.command == "get":
        get_speaker(args.qdrant, args.collection, args.id)
    elif args.command == "rename":
        rename_speaker(args.qdrant, args.collection, args.id, args.new_name)
    elif args.command == "delete":
        delete_speaker(args.qdrant, args.collection, args.id)
`
	return os.WriteFile(scriptPath, []byte(content), 0755)
}

// ListSpeakers retrieves all speakers from the vector DB
func (t *TitanetAdapter) ListSpeakers(ctx context.Context) ([]SpeakerInfo, error) {
	if err := t.EnsureManagementScript(); err != nil {
		return nil, err
	}

	qdrantHost := os.Getenv("QDRANT_HOST")
	if qdrantHost == "" {
		qdrantHost = "qdrant"
	}

	scriptPath := filepath.Join(t.envPath, "titanet_manage.py")
	cmd := exec.CommandContext(ctx, "uv", "run", "--native-tls", "--project", t.envPath, "python", scriptPath,
		"list",
		"--qdrant", qdrantHost,
	)

	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to list speakers: %w", err)
	}

	var speakers []SpeakerInfo
	if err := json.Unmarshal(output, &speakers); err != nil {
		return nil, fmt.Errorf("failed to parse speakers list: %w", err)
	}

	return speakers, nil
}

// GetSpeaker retrieves a single speaker from the vector DB
func (t *TitanetAdapter) GetSpeaker(ctx context.Context, id string) (*SpeakerInfo, error) {
	if err := t.EnsureManagementScript(); err != nil {
		return nil, err
	}

	qdrantHost := os.Getenv("QDRANT_HOST")
	if qdrantHost == "" {
		qdrantHost = "qdrant"
	}

	scriptPath := filepath.Join(t.envPath, "titanet_manage.py")
	cmd := exec.CommandContext(ctx, "uv", "run", "--native-tls", "--project", t.envPath, "python", scriptPath,
		"get",
		id,
		"--qdrant", qdrantHost,
	)

	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("failed to get speaker: %s", string(output))
	}

	var speaker SpeakerInfo
	if err := json.Unmarshal(output, &speaker); err != nil {
		return nil, fmt.Errorf("failed to parse speaker info: %w", err)
	}

	return &speaker, nil
}

// RenameSpeaker updates a speaker's name
func (t *TitanetAdapter) RenameSpeaker(ctx context.Context, id, newName string) error {
	if err := t.EnsureManagementScript(); err != nil {
		return err
	}

	qdrantHost := os.Getenv("QDRANT_HOST")
	if qdrantHost == "" {
		qdrantHost = "qdrant"
	}

	scriptPath := filepath.Join(t.envPath, "titanet_manage.py")
	cmd := exec.CommandContext(ctx, "uv", "run", "--native-tls", "--project", t.envPath, "python", scriptPath,
		"rename",
		id,
		newName,
		"--qdrant", qdrantHost,
	)

	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to rename speaker: %s", string(output))
	}

	return nil
}

// DeleteSpeaker removes a speaker

func (t *TitanetAdapter) DeleteSpeaker(ctx context.Context, id string) error {

	if err := t.EnsureManagementScript(); err != nil {

		return err

	}



	qdrantHost := os.Getenv("QDRANT_HOST")

	if qdrantHost == "" {

		qdrantHost = "qdrant"

	}



	scriptPath := filepath.Join(t.envPath, "titanet_manage.py")

	cmd := exec.CommandContext(ctx, "uv", "run", "--native-tls", "--project", t.envPath, "python", scriptPath,

		"delete",

		id,

		"--qdrant", qdrantHost,

	)



	if output, err := cmd.CombinedOutput(); err != nil {

		return fmt.Errorf("failed to delete speaker: %s", string(output))

	}



	return nil

}



// RefreshSnormCohort runs the cohort refresh script.

func (t *TitanetAdapter) RefreshSnormCohort(ctx context.Context) error {

	qdrantHost := os.Getenv("QDRANT_HOST")

	if qdrantHost == "" {

		qdrantHost = "qdrant"

	}



	scriptPath := filepath.Join("data/whisperx-env/parakeet", "titanet_cohort_manager.py")

	cmd := exec.CommandContext(ctx, "uv", "run", "--native-tls", "--project", t.envPath, "python", scriptPath,

		"--qdrant", qdrantHost,

	)



	logger.Info("Executing S-Norm cohort refresh command", "args", strings.Join(cmd.Args, " "))



	output, err := cmd.CombinedOutput()

	if err != nil {

		logger.Error("S-Norm cohort refresh script failed", "output", string(output))

		return fmt.Errorf("s-norm cohort refresh script failed: %w", err)

	}



	logger.Info("S-Norm cohort refresh successful", "output", string(output))

	return nil

}
