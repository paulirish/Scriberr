"""Tests for parakeet_transcribe_buffered.py"""
import pytest
from pathlib import Path


def test_parakeet_transcribe_buffered_exists():
    """Verify parakeet_transcribe_buffered.py exists."""
    script_path = Path(__file__).parent.parent / "parakeet_transcribe_buffered.py"
    assert script_path.exists(), "parakeet_transcribe_buffered.py should exist"


def test_parakeet_transcribe_buffered_is_readable():
    """Verify parakeet_transcribe_buffered.py is readable."""
    script_path = Path(__file__).parent.parent / "parakeet_transcribe_buffered.py"
    content = script_path.read_text()
    assert len(content) > 0, "parakeet_transcribe_buffered.py should not be empty"


def test_parakeet_transcribe_buffered_has_main():
    """Verify parakeet_transcribe_buffered.py has main execution block."""
    script_path = Path(__file__).parent.parent / "parakeet_transcribe_buffered.py"
    content = script_path.read_text()
    assert "if __name__" in content, "Script should have main execution block"


def test_parakeet_transcribe_buffered_imports():
    """Verify parakeet_transcribe_buffered.py has expected imports."""
    script_path = Path(__file__).parent.parent / "parakeet_transcribe_buffered.py"
    content = script_path.read_text()
    # Check for NeMo imports and buffered inference
    assert "nemo" in content.lower(), "Script should import NeMo"
    assert "buffer" in content.lower(), "Script should reference buffering"
