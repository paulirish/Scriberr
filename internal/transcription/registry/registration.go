package registry

import (
	"path/filepath"
	"scriberr/internal/config"
	"scriberr/internal/transcription/adapters"
	"scriberr/pkg/logger"
)

// RegisterStandardAdapters registers all built-in model adapters using the provided configuration.
// This centralizes adapter registration so it can be used by the server, CLI, and setup tools.
func RegisterStandardAdapters(cfg *config.Config) {
	// Shared environment path for NVIDIA models (NeMo-based)
	nvidiaEnvPath := filepath.Join(cfg.WhisperXEnv, "parakeet")
	
	// Dedicated environment path for PyAnnote (to avoid dependency conflicts)
	pyannoteEnvPath := filepath.Join(cfg.WhisperXEnv, "pyannote")

	logger.Info("Registering standard adapters", 
		"nvidia_env", nvidiaEnvPath, 
		"pyannote_env", pyannoteEnvPath)

	// Register transcription adapters
	RegisterTranscriptionAdapter("parakeet",
		adapters.NewParakeetAdapter(nvidiaEnvPath))
	RegisterTranscriptionAdapter("canary",
		adapters.NewCanaryAdapter(nvidiaEnvPath))
	RegisterTranscriptionAdapter("openai_whisper",
		adapters.NewOpenAIAdapter(cfg.OpenAIAPIKey))

	// Register diarization adapters
	RegisterDiarizationAdapter("sortformer",
		adapters.NewSortformerAdapter(nvidiaEnvPath))
	
	// PyAnnote is registered here so it's available in the setup tool and server
	RegisterDiarizationAdapter("pyannote",
		adapters.NewPyAnnoteAdapter(pyannoteEnvPath))

	// Register speaker identification adapters
	RegisterIdentificationAdapter("titanet",
		adapters.NewTitanetAdapter(nvidiaEnvPath))

	logger.Info("Standard adapter registration complete")
}
