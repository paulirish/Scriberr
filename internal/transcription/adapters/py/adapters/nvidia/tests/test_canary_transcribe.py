"""Tests for canary_transcribe.py"""
import pytest
import subprocess
import json
import os
from pathlib import Path

# Paths
SCRIPT_DIR = Path(__file__).parent.parent
TEST_DATA_DIR = Path(__file__).parent.parent.parent.parent.parent.parent.parent.parent / "tests/data"
AUDIO_FILE = TEST_DATA_DIR / "AMI-Corpus-IB4002.Mix-Headset-clip.wav"

def test_canary_transcription_output():
    """Verify Canary transcription output matches expected results."""
    
    if not AUDIO_FILE.exists():
        pytest.skip(f"Audio file not found: {AUDIO_FILE}")

    # Locate project root and paths
    project_root = Path(__file__).resolve().parents[7] 
    env_path = project_root / "data/whisperx-env/parakeet" # Canary uses the same env as Parakeet in this setup
    script_path = SCRIPT_DIR / "canary_transcribe.py"
    
    if not env_path.exists():
        pytest.skip(f"Environment not found at {env_path}")
        
    cmd = [
        "uv", "run", 
        "--project", str(env_path),
        "python", str(script_path),
        str(AUDIO_FILE),
        "--timestamps"
    ]
    
    print(f"Running command: {' '.join(cmd)}")
    
    result = subprocess.run(
        cmd, 
        capture_output=True, 
        text=True, 
        cwd=project_root
    )
    
    if result.returncode != 0:
        pytest.fail(f"Script failed with error:\n{result.stderr}")
        
    # Extract JSON output
    output_lines = result.stdout.strip().split('\n')
    json_str = ""
    
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
    assert data["source_language"] == "en"
    assert data["target_language"] == "en"
    assert data["task"] == "transcribe"
    assert data["model"] == "canary-1b-v2"
    
    # Canary output text check
    assert "Most of us" in data["transcription"]
    assert "desktop computers" in data["transcription"]
    
    assert "word_timestamps" in data
    assert len(data["word_timestamps"]) > 0
    # Check first word
    first_word = data["word_timestamps"][0]
    assert first_word["word"] == "Most"
    # Allow for small floating point differences
    assert abs(first_word["start"] - 0.0) < 0.1
    
    assert "segment_timestamps" in data
    assert len(data["segment_timestamps"]) > 0