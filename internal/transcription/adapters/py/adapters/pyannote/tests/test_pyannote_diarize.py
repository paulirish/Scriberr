"""Tests for pyannote_diarize.py"""
import pytest
from pathlib import Path


def test_pyannote_diarize_exists():
    """Verify pyannote_diarize.py exists."""
    script_path = Path(__file__).parent.parent / "pyannote_diarize.py"
    assert script_path.exists(), "pyannote_diarize.py should exist"


def test_pyannote_diarize_is_readable():
    """Verify pyannote_diarize.py is readable."""
    script_path = Path(__file__).parent.parent / "pyannote_diarize.py"
    content = script_path.read_text()
    assert len(content) > 0, "pyannote_diarize.py should not be empty"


def test_pyannote_diarize_has_main():
    """Verify pyannote_diarize.py has main execution block."""
    script_path = Path(__file__).parent.parent / "pyannote_diarize.py"
    content = script_path.read_text()
    assert "if __name__" in content, "Script should have main execution block"


def test_pyannote_diarize_imports():
    """Verify pyannote_diarize.py has expected imports."""
    script_path = Path(__file__).parent.parent / "pyannote_diarize.py"
    content = script_path.read_text()
    # Check for PyAnnote imports
    assert "pyannote" in content.lower(), "Script should import PyAnnote"


def test_pyannote_diarize_has_pipeline():
    """Verify pyannote_diarize.py references Pipeline."""
    script_path = Path(__file__).parent.parent / "pyannote_diarize.py"
    content = script_path.read_text()
    assert "Pipeline" in content, "Script should use PyAnnote Pipeline"
