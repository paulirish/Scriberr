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
		Version:            "1.0.0",
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
			Description: "Threshold for cosine similarity to identify a speaker",
		},
		{
			Name:        "alpha_max",
			Type:        "float",
			Default:     0.25,
			Description: "Maximum learning rate for Confidence-Weighted EMA.",
		},
		{
			Name:        "min_duration_for_full_weight",
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

	// Dependency check (qdrant-client) is handled by the shared environment setup in SortformerAdapter
	// But we should ensure the specific model is downloaded
	if err := t.downloadTitanetModel(); err != nil {
		return fmt.Errorf("failed to download TitaNet model: %w", err)
	}

	// Create identification script
	if err := t.createIdentityScript(); err != nil {
		return fmt.Errorf("failed to create identity script: %w", err)
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
	// URL for TitaNet Large from NGC or HuggingFace
	modelURL := "https://huggingface.co/nvidia/speakerverification_en_titanet_large/resolve/main/speakerverification_en_titanet_large.nemo?download=true"

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
	defer cancel()

	return downloader.DownloadFile(ctx, modelURL, modelPath)
}

func (t *TitanetAdapter) createIdentityScript() error {
	scriptPath := filepath.Join(t.envPath, "titanet_identify.py")
	if _, err := os.Stat(scriptPath); err == nil {
		return nil
	}

	// Python script content
	content := `#!/usr/bin/env python3
"""
TitaNet Speaker Identification Script (v2 with Confidence-Weighted EMA)
"""
import argparse
import json
import os
import sys
import torch
import numpy as np
from typing import List, Dict
import logging

# Setup logging
logging.basicConfig(level=logging.INFO, format='%(asctime)s - %(levelname)s - %(message)s')
logger = logging.getLogger(__name__)

try:
    from nemo.collections.asr.models import EncDecSpeakerLabelModel
    from qdrant_client import QdrantClient
    from qdrant_client.http import models as qmodels
except ImportError as e:
    logger.error(f"Import Error: {e}")
    sys.exit(1)

def setup_qdrant(host: str, collection_name: str, vector_size: int = 192):
    client = QdrantClient(host=host, port=6333)
    collections = client.get_collections().collections
    exists = any(c.name == collection_name for c in collections)
    if not exists:
        logger.info(f"Creating collection {collection_name}")
        client.create_collection(
            collection_name=collection_name,
            vectors_config=qmodels.VectorParams(size=vector_size, distance=qmodels.Distance.COSINE),
        )
    return client

def identify_speakers(
    audio_path: str,
    segments_file: str,
    output_file: str,
    qdrant_host: str = "qdrant",
    collection_name: str = "speakers",
    threshold: float = 0.7,
    alpha_max: float = 0.25,
    min_duration_for_full_weight: float = 4.0,
    device: str = "auto"
):
    # 1. Load Model
    if device == "auto":
        device = "cuda" if torch.cuda.is_available() else "cpu"
    logger.info(f"Loading TitaNet on {device}")
    script_dir = os.path.dirname(os.path.abspath(__file__))
    model_path = os.path.join(script_dir, "titanet-l.nemo")
    model = EncDecSpeakerLabelModel.restore_from(restore_path=model_path, map_location=device)
    model.eval()

    # 2. Setup Qdrant
    client = setup_qdrant(qdrant_host, collection_name, vector_size=192)

    # 3. Load Segments
    with open(segments_file, 'r') as f:
        data = json.load(f)
        segments = data.get("segments", [])

    local_speakers = {}
    for idx, seg in enumerate(segments):
        spk = seg.get("speaker")
        if spk not in local_speakers:
            local_speakers[spk] = []
        local_speakers[spk].append(idx)

    # 4. Process each local speaker
    global_mapping = {}
    import soundfile as sf
    full_waveform, sample_rate = sf.read(audio_path)
    full_waveform = torch.from_numpy(full_waveform).float()
    if full_waveform.ndim > 1: full_waveform = full_waveform.mean(dim=0)
    full_waveform = full_waveform.unsqueeze(0).to(device)

    for local_spk, indices in local_speakers.items():
        logger.info(f"Processing local speaker {local_spk} ({len(indices)} segments)")
        embeddings = []
        indices.sort(key=lambda i: segments[i].get("end") - segments[i].get("start"), reverse=True)
        top_indices = indices[:10]
        
        total_duration = 0
        for idx in top_indices:
            seg = segments[idx]
            start, end = seg["start"], seg["end"]
            duration = end - start
            total_duration += duration

            start_frame, end_frame = int(start * sample_rate), int(end * sample_rate)
            if end_frame > full_waveform.shape[1]: end_frame = full_waveform.shape[1]
            sub_audio = full_waveform[:, start_frame:end_frame]
            
            if sub_audio.shape[1] < 1600: continue
            len_tensor = torch.tensor([sub_audio.shape[1]], device=device)
            with torch.no_grad():
                _, embs = model(input_signal=sub_audio, input_signal_length=len_tensor)
                embeddings.append(embs[0].cpu().numpy())

        if not embeddings:
            logger.warning(f"No valid embeddings for {local_spk}")
            continue

        centroid = np.mean(embeddings, axis=0)
        norm = np.linalg.norm(centroid)
        if norm > 0: centroid = centroid / norm

        # 5. Query Qdrant
        search_result = client.search(
            collection_name=collection_name,
            query_vector=centroid.tolist(),
            limit=1,
            score_threshold=threshold,
            with_vector=True  # Retrieve vector for EMA update
        )

        if search_result:
            # Match found - Update profile with Confidence-Weighted EMA
            best_match = search_result[0]
            global_id = best_match.payload.get("name", "Unknown")
            logger.info(f"Matched {local_spk} to {global_id} (score: {best_match.score:.4f})")

            existing_centroid = np.array(best_match.vector)
            wt = min(alpha_max, total_duration / min_duration_for_full_weight)
            
            logger.info(f"Updating speaker {global_id} with weight {wt:.4f} from {total_duration:.2f}s of audio")
            updated_centroid = ((1 - wt) * existing_centroid) + (wt * centroid)
            
            norm = np.linalg.norm(updated_centroid)
            if norm > 0: updated_centroid = updated_centroid / norm
            
            client.upsert(
                collection_name=collection_name,
                points=[qmodels.PointStruct(id=best_match.id, vector=updated_centroid.tolist(), payload=best_match.payload)]
            )
        else:
            # No match -> Enroll new speaker
            import uuid
            new_id = str(uuid.uuid4())
            human_name = f"Speaker-{new_id[:8]}"
            logger.info(f"Enrolling {local_spk} as new speaker {human_name}")
            client.upsert(
                collection_name=collection_name,
                points=[
                    qmodels.PointStruct(
                        id=new_id,
                        vector=centroid.tolist(),
                        payload={"name": human_name, "created_at": str(os.path.getctime(audio_path))}
                    )
                ]
            )
            global_id = human_name

        global_mapping[local_spk] = global_id

    # 6. Update Segments and Save
    for seg in segments:
        local = seg.get("speaker")
        if local in global_mapping:
            seg["speaker"] = global_mapping[local]
    
    output_data = {"segments": segments}
    with open(output_file, 'w') as f:
        json.dump(output_data, f, indent=2)

    logger.info(f"Identification complete. Saved to {output_file}")

if __name__ == "__main__":
    parser = argparse.ArgumentParser(description="TitaNet Speaker Identification")
    parser.add_argument("audio_file")
    parser.add_argument("segments_file")
    parser.add_argument("output_file")
    parser.add_argument("--qdrant", default="qdrant")
    parser.add_argument("--collection", default="speakers")
    parser.add_argument("--threshold", type=float, default=0.7)
    parser.add_argument("--alpha-max", type=float, default=0.25, help="Max learning rate for EMA.")
    parser.add_argument("--min-duration-full-weight", type=float, default=4.0, help="Min duration for full weight in EMA.")
    args = parser.parse_args()

    identify_speakers(
        args.audio_file,
        args.segments_file,
        args.output_file,
        qdrant_host=args.qdrant,
        collection_name=args.collection,
        threshold=args.threshold,
        alpha_max=args.alpha_max,
        min_duration_for_full_weight=args.min_duration_full_weight,
    )
`
	return os.WriteFile(scriptPath, []byte(content), 0755)
}

// IdentifySpeakers runs the identification process
func (t *TitanetAdapter) IdentifySpeakers(ctx context.Context, input interfaces.AudioInput, diarizationResult *interfaces.DiarizationResult, params map[string]interface{}, procCtx interfaces.ProcessingContext) (*interfaces.DiarizationResult, error) {
	// 1. Serialize current diarization result to a temp JSON file
	tempDir, err := t.CreateTempDirectory(procCtx)
	if err != nil {
		return nil, err
	}
	defer t.CleanupTempDirectory(tempDir)

	inputJSON := filepath.Join(tempDir, "input_segments.json")
	outputJSON := filepath.Join(tempDir, "output_segments.json")

	// Wrap in expected JSON structure
	wrapper := map[string]interface{}{
		"segments": diarizationResult.Segments,
	}

	data, _ := json.Marshal(wrapper)
	if err := os.WriteFile(inputJSON, data, 0644); err != nil {
		return nil, fmt.Errorf("failed to write input json: %w", err)
	}

	// 2. Build command
	scriptPath := filepath.Join(t.envPath, "titanet_identify.py")
	// Use Qdrant host from env or default
	qdrantHost := os.Getenv("QDRANT_HOST")
	if qdrantHost == "" {
		qdrantHost = "qdrant" // Default in docker-compose
	}

	cmd := exec.CommandContext(ctx, "uv", "run", "--native-tls", "--project", t.envPath, "python", scriptPath,
		input.FilePath,
		inputJSON,
		outputJSON,
		"--qdrant", qdrantHost,
		"--threshold", fmt.Sprintf("%.2f", t.GetFloatParameter(params, "similarity_threshold")),
		"--alpha-max", fmt.Sprintf("%.2f", t.GetFloatParameter(params, "alpha_max")),
		"--min-duration-full-weight", fmt.Sprintf("%.2f", t.GetFloatParameter(params, "min_duration_for_full_weight")),
	)

  logger.Info("Executing Titanet command", "args", strings.Join(cmd.Args, " "))


	cmd.Env = append(os.Environ(), "PYTHONUNBUFFERED=1")
    // Capture output for debugging
	output, err := cmd.CombinedOutput()
	if err != nil {
		logger.Error("Identity script failed", "output", string(output))
		return nil, fmt.Errorf("identity script failed: %w", err)
	}
	logger.Debug("Identity script output", "output", string(output))

	// Extract speaker matching/enrolling events from the log
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

	// 3. Read back result
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

	// Update the passed result object or return new one
	newResult := *diarizationResult
	newResult.Segments = resultWrapper.Segments

	// Re-calculate unique speakers
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
        # Scroll through points to get all speakers
        # Note: This might be slow for huge collections, but fine for <10k speakers
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

func get_speaker(host: str, collection: str, speaker_id: str):
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
        # Verify existence
        points = client.retrieve(
            collection_name=collection,
            ids=[speaker_id]
        )

        if not points:
            logger.error("Speaker not found")
            sys.exit(1)

        # Update payload
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

    # List
    parser_list = subparsers.add_parser("list")
    parser_list.add_argument("--qdrant", default="qdrant")
    parser_list.add_argument("--collection", default="speakers")

    # Get
    parser_get = subparsers.add_parser("get")
    parser_get.add_argument("id")
    parser_get.add_argument("--qdrant", default="qdrant")
    parser_get.add_argument("--collection", default="speakers")

    # Rename
    parser_rename = subparsers.add_parser("rename")
    parser_rename.add_argument("id")
    parser_rename.add_argument("new_name")
    parser_rename.add_argument("--qdrant", default="qdrant")
    parser_rename.add_argument("--collection", default="speakers")

    # Delete
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
