"""Tests for sortformer_diarize.py"""
import pytest
from pathlib import Path


def test_sortformer_diarize_exists():
    """Verify sortformer_diarize.py exists."""
    script_path = Path(__file__).parent.parent / "sortformer_diarize.py"
    assert script_path.exists(), "sortformer_diarize.py should exist"


def test_sortformer_diarize_is_readable():
    """Verify sortformer_diarize.py is readable."""
    script_path = Path(__file__).parent.parent / "sortformer_diarize.py"
    content = script_path.read_text()
    assert len(content) > 0, "sortformer_diarize.py should not be empty"


def test_sortformer_diarize_has_main():
    """Verify sortformer_diarize.py has main execution block."""
    script_path = Path(__file__).parent.parent / "sortformer_diarize.py"
    content = script_path.read_text()
    assert "if __name__" in content, "Script should have main execution block"


def test_sortformer_diarize_imports():
    """Verify sortformer_diarize.py has expected imports."""
    script_path = Path(__file__).parent.parent / "sortformer_diarize.py"
    content = script_path.read_text()
    # Check for NeMo imports and diarization
    assert "nemo" in content.lower(), "Script should import NeMo"
    assert "diariz" in content.lower(), "Script should reference diarization"
