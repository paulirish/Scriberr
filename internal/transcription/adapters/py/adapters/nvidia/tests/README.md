# NVIDIA Adapter Tests

This directory contains tests for the NVIDIA adapter scripts (Canary, Parakeet, Sortformer).

## Running Tests

### Option 1: Using uv (Recommended)

From this directory:
```bash
cd internal/transcription/adapters/py/adapters/nvidia
uv sync --extra dev
uv run pytest tests/ -v
```

### Option 2: From project root

```bash
uv run --project internal/transcription/adapters/py/adapters/nvidia pytest internal/transcription/adapters/py/adapters/nvidia/tests/ -v
```

### Option 3: Using system Python (if pytest is installed)

```bash
cd internal/transcription/adapters/py/adapters/nvidia
python -m pytest tests/ -v
```

## Test Structure

- `test_canary_transcribe.py` - Tests for Canary transcription script
- `test_parakeet_transcribe.py` - Tests for Parakeet transcription script
- `test_parakeet_transcribe_buffered.py` - Tests for Parakeet buffered inference
- `test_sortformer_diarize.py` - Tests for Sortformer diarization script

## Current Tests

The current tests are basic structural tests that verify:
- Scripts exist
- Scripts are readable
- Scripts have expected structure (main block, imports)

## Adding More Tests

To add more comprehensive tests, you can:
1. Mock the NeMo dependencies
2. Test argument parsing
3. Test output formatting
4. Add integration tests with sample audio files
