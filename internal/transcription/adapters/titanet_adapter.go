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
			Default:     0.5,
			Description: "Threshold for cosine similarity to identify a speaker",
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

	// Copy identification script
	if err := t.copyIdentifyScript(); err != nil {
		return fmt.Errorf("failed to copy identity script: %w", err)
	}

	if err := t.EnsureManagementScript(); err != nil {
		return fmt.Errorf("failed to ensure management script: %w", err)
	}

	// Check if environment is already ready
	if CheckEnvironmentReady(t.envPath, "import nemo") {
		// Ensure model is downloaded
		if err := t.downloadTitanetModel(); err != nil {
			return fmt.Errorf("failed to download TitaNet model: %w", err)
		}
		t.initialized = true
		return nil
	}

	// Dependency check (qdrant-client) is handled by the shared environment setup in SortformerAdapter
	// but we'll trigger a sync via whatever adapter hits this first if needed.
	// For Titanet, we don't have a dedicated setup function yet because it shares with Sortformer/Parakeet.
	// If we get here, it means nemo isn't importable, so we should probably fail or wait for another adapter.
	// Since they all run in parallel, Sortformer or Parakeet will likely handle the uv sync.
	
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

func (t *TitanetAdapter) copyIdentifyScript() error {
  scriptContent, err := nvidiaScripts.ReadFile("py/nvidia/titanet_identify.py")
	if err != nil {
		return fmt.Errorf("failed to read embedded titanet_identify.py: %w", err)
	}

	scriptPath := filepath.Join(t.envPath, "titanet_identify.py")
	if err := os.WriteFile(scriptPath, scriptContent, 0755); err != nil {
		return fmt.Errorf("failed to write identify script: %w", err)
	}

	return nil
}

// EnsureManagementScript copies the python script for managing speakers
func (t *TitanetAdapter) EnsureManagementScript() error {
  scriptContent, err := nvidiaScripts.ReadFile("py/nvidia/titanet_manage.py")
	if err != nil {
		return fmt.Errorf("failed to read embedded titanet_manage.py: %w", err)
	}

	scriptPath := filepath.Join(t.envPath, "titanet_manage.py")
	if err := os.WriteFile(scriptPath, scriptContent, 0755); err != nil {
		return fmt.Errorf("failed to write manage script: %w", err)
	}

	return nil
}

func (t *TitanetAdapter) getQdrantHost() string {
	if qdrantHost := os.Getenv("QDRANT_HOST"); qdrantHost != "" {
		return qdrantHost
	}

	// If QDRANT_HOST is not set, check if we're running in Docker
	if _, err := os.Stat("/.dockerenv"); err == nil {
		return "qdrant"
	}

	return "localhost"
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
	qdrantHost := t.getQdrantHost()

	cmd := exec.CommandContext(ctx, "uv", "run", "--native-tls", "--project", t.envPath, "python", scriptPath,
		input.FilePath,
		inputJSON,
		outputJSON,
		"--qdrant", qdrantHost,
		"--threshold", fmt.Sprintf("%.2f", t.GetFloatParameter(params, "similarity_threshold")),
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

// ListSpeakers retrieves all speakers from the vector DB
func (t *TitanetAdapter) ListSpeakers(ctx context.Context) ([]interfaces.SpeakerInfo, error) {
	qdrantHost := t.getQdrantHost()

	scriptPath := filepath.Join(t.envPath, "titanet_manage.py")
	cmd := exec.CommandContext(ctx, "uv", "run", "--native-tls", "--project", t.envPath, "python", scriptPath,
		"list",
		"--qdrant", qdrantHost,
	)

	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("failed to list speakers: %w (output: %s)", err, string(output))
	}

	var speakers []interfaces.SpeakerInfo
	if err := json.Unmarshal(output, &speakers); err != nil {
		return nil, fmt.Errorf("failed to parse speakers list: %w (output: %s)", err, string(output))
	}

	return speakers, nil
}

// GetSpeaker retrieves a single speaker from the vector DB
func (t *TitanetAdapter) GetSpeaker(ctx context.Context, id string) (*interfaces.SpeakerInfo, error) {
	qdrantHost := t.getQdrantHost()

	scriptPath := filepath.Join(t.envPath, "titanet_manage.py")
	cmd := exec.CommandContext(ctx, "uv", "run", "--native-tls", "--project", t.envPath, "python", scriptPath,
		"get",
		id,
		"--qdrant", qdrantHost,
	)

	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("failed to get speaker: %w (output: %s)", err, string(output))
	}

	var speaker interfaces.SpeakerInfo
	if err := json.Unmarshal(output, &speaker); err != nil {
		return nil, fmt.Errorf("failed to parse speaker info: %w (output: %s)", err, string(output))
	}

	return &speaker, nil
}

// RenameSpeaker updates a speaker's name
func (t *TitanetAdapter) RenameSpeaker(ctx context.Context, id, newName string) error {
	qdrantHost := t.getQdrantHost()

	scriptPath := filepath.Join(t.envPath, "titanet_manage.py")
	cmd := exec.CommandContext(ctx, "uv", "run", "--native-tls", "--project", t.envPath, "python", scriptPath,
		"rename",
		id,
		newName,
		"--qdrant", qdrantHost,
	)

	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to rename speaker: %w (output: %s)", err, string(output))
	}

	return nil
}

// DeleteSpeaker removes a speaker
func (t *TitanetAdapter) DeleteSpeaker(ctx context.Context, id string) error {
	qdrantHost := t.getQdrantHost()

	scriptPath := filepath.Join(t.envPath, "titanet_manage.py")
	cmd := exec.CommandContext(ctx, "uv", "run", "--native-tls", "--project", t.envPath, "python", scriptPath,
		"delete",
		id,
		"--qdrant", qdrantHost,
	)

	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to delete speaker: %w (output: %s)", err, string(output))
	}

	return nil
}
