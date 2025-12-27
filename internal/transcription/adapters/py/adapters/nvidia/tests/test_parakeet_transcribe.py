"""Tests for parakeet_transcribe.py"""
import pytest
import subprocess
import json
import os
from pathlib import Path

# Paths
SCRIPT_DIR = Path(__file__).parent.parent
TEST_DATA_DIR = Path(__file__).parent.parent.parent.parent.parent.parent.parent.parent / "tests/data"
AUDIO_FILE = TEST_DATA_DIR / "AMI-Corpus-IB4002.Mix-Headset-clip.wav"

def test_parakeet_transcription_output():
    """Verify Parakeet transcription output matches expected results."""
    
    if not AUDIO_FILE.exists():
        pytest.skip(f"Audio file not found: {AUDIO_FILE}")

    # Construct command
    # uv run --project data/whisperx-env/parakeet python internal/transcription/adapters/py/adapters/nvidia/parakeet_transcribe.py ...
    
    # We need to find the project root to construct the path relative to it or use absolute paths
    # Assuming the test is run from the project root or we can find the whisperx-env
    
    # Locate project root (Scriberr directory)
    # This file is in internal/transcription/adapters/py/adapters/nvidia/tests/
    project_root = Path(__file__).resolve().parents[7] 
    env_path = project_root / "data/whisperx-env/parakeet"
    script_path = SCRIPT_DIR / "parakeet_transcribe.py"
    
    if not env_path.exists():
        pytest.skip(f"Environment not found at {env_path}")
        
    cmd = [
        "uv", "run", 
        "--project", str(env_path),
        "python", str(script_path),
        str(AUDIO_FILE),
        "--timestamps",
        "--context-left", "256",
        "--context-right", "256"
    ]
    
    print(f"Running command: {' '.join(cmd)}")
    
    result = subprocess.run(
        cmd, 
        capture_output=True, 
        text=True, 
        cwd=project_root # Run from project root to ensure paths are correct if relative
    )
    
    if result.returncode != 0:
        pytest.fail(f"Script failed with error:\n{result.stderr}")
        
    # The output contains logs + JSON. We need to extract the JSON part.
    # The script prints JSON at the end (or saves to file if --output is used)
    # Since we didn't use --output, we look for the last valid JSON object in stdout
    
    output_lines = result.stdout.strip().split('\n')
    json_str = ""
    
    # Find the start of the JSON output (starts with {)
    for i in range(len(output_lines) - 1, -1, -1):
        if output_lines[i].strip().startswith("{"):
            json_str = "\n".join(output_lines[i:])
            break
            
    assert json_str, "Could not find JSON output in stdout"
    
    try:
        data = json.loads(json_str)
    except json.JSONDecodeError as e:
        pytest.fail(f"Failed to parse JSON output: {e}\nOutput was:\n{json_str}")
        
    # Assertions based on the provided sample output
    assert data["language"] == "en"
    assert data["model"] == "parakeet-tdt-0.6b-v3"
    assert "transcription" in data
    assert "First of all, have desktop computers" in data["transcription"]
    assert "reading room" in data["transcription"]
    
    assert "word_timestamps" in data
    assert len(data["word_timestamps"]) > 0
    # Check first word
    first_word = data["word_timestamps"][0]
    assert first_word["word"] == "First"
    # Allow for small floating point differences
    assert abs(first_word["start"] - 0.4) < 0.1
    
    assert "segment_timestamps" in data
    assert len(data["segment_timestamps"]) > 0
    
    assert data["context"]["left"] == 256
    assert data["context"]["right"] == 256