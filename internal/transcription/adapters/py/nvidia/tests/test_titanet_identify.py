"""Tests for titanet_identify.py"""

import pytest
import subprocess
import json
import os
import tempfile
from pathlib import Path

# Paths
SCRIPT_DIR = Path(__file__).parent.parent
TEST_DATA_DIR = (
    Path(__file__).parent.parent.parent.parent.parent.parent.parent / "tests/data"
)
AUDIO_FILE = TEST_DATA_DIR / "AMI-Corpus-IB4002.Mix-Headset-clip.wav"
SEGMENTS_FILE = TEST_DATA_DIR / "sf-segments.json"


def test_titanet_identification_output():
    """Verify TitaNet identification output matches expected results."""

    assert AUDIO_FILE.exists(), f"Audio file not found: {AUDIO_FILE}"
    # SEGMENTS_FILE might be ignored by git/gemini but should exist on disk
    assert SEGMENTS_FILE.exists(), f"Segments file not found: {SEGMENTS_FILE}"

    # Locate project root and paths
    project_root = Path(__file__).resolve().parents[6]
    env_path = project_root / "data/whisperx-env/parakeet"
    script_path = SCRIPT_DIR / "titanet_identify.py"

    assert env_path.exists(), f"Environment not found at: {env_path}"

    # Create a temporary file for output
    with tempfile.NamedTemporaryFile(suffix=".json", delete=False) as tmp_file:
        output_file = tmp_file.name

    try:
        cmd = [
            "uv",
            "run",
            "--project",
            str(env_path),
            "python",
            str(script_path),
            str(AUDIO_FILE),
            str(SEGMENTS_FILE),
            output_file,
            "--qdrant",
            "mock",
        ]

        print(f"Running command: {' '.join(cmd)}")

        result = subprocess.run(cmd, capture_output=True, text=True, cwd=project_root)

        if result.returncode != 0:
            pytest.fail(
                f"Script failed with error:\n{result.stderr}\nStdout:\n{result.stdout}"
            )

        # Verify output file exists and is valid JSON
        assert os.path.exists(output_file), "Output file was not created"

        with open(output_file, "r") as f:
            data = json.load(f)

        # Assertions
        assert data["model"] == "nvidia/diar_streaming_sortformer_4spk-v2"
        assert "segments" in data
        assert len(data["segments"]) > 0

        # Check first segment structure
        first_segment = data["segments"][0]
        assert "start" in first_segment
        assert "end" in first_segment

        # titanet_identify adds 'original_speaker' and updates 'speaker'
        assert "speaker" in first_segment
        assert "original_speaker" in first_segment
        assert first_segment["original_speaker"] == "speaker_0"

        # In mock mode with no prior collection, it enrolls new speakers
        # The new speaker name usually starts with "Spk-" or similar if using the logic in script
        # The script uses human_name = f"Spk-{new_id[:8]}"
        assert str(first_segment["speaker"]).startswith("Spk-")

        # Verify that all segments have been processed
        for seg in data["segments"]:
            assert "original_speaker" in seg
            assert str(seg["speaker"]).startswith("Spk-")

        # Check that we have a mapping that is consistent
        # i.e. all segments with same original_speaker should have same new speaker
        speaker_map = {}
        for seg in data["segments"]:
            orig = seg["original_speaker"]
            new = seg["speaker"]
            if orig in speaker_map:
                assert speaker_map[orig] == new, (
                    f"Inconsistent mapping for {orig}: {speaker_map[orig]} vs {new}"
                )
            else:
                speaker_map[orig] = new

    finally:
        # Cleanup
        if os.path.exists(output_file):
            os.remove(output_file)
