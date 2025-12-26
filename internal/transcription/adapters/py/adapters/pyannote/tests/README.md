# PyAnnote Adapter Tests

This directory contains tests for the PyAnnote diarization adapter script.

## Running Tests

### Option 1: Using uv (Recommended)

From this directory:
```bash
cd internal/transcription/adapters/py/adapters/pyannote
uv sync --extra dev
uv run pytest tests/ -v
```

### Option 2: From project root

```bash
uv run --project internal/transcription/adapters/py/adapters/pyannote pytest internal/transcription/adapters/py/adapters/pyannote/tests/ -v
```

### Option 3: Using system Python (if pytest is installed)

```bash
cd internal/transcription/adapters/py/adapters/pyannote
python -m pytest tests/ -v
```

## Test Structure

- `test_pyannote_diarize.py` - Tests for PyAnnote diarization script

## Current Tests

The current tests are basic structural tests that verify:
- Script exists
- Script is readable
- Script has expected structure (main block, imports, Pipeline usage)

## Adding More Tests

To add more comprehensive tests, you can:
1. Mock the PyAnnote dependencies
2. Test argument parsing
3. Test output formatting (JSON and RTTM)
4. Add integration tests with sample audio files
5. Test HuggingFace token handling
