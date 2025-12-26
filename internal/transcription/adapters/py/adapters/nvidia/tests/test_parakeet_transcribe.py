"""Tests for parakeet_transcribe.py"""
import pytest
from pathlib import Path


def test_parakeet_transcribe_exists():
    """Verify parakeet_transcribe.py exists."""
    script_path = Path(__file__).parent.parent / "parakeet_transcribe.py"
    assert script_path.exists(), "parakeet_transcribe.py should exist"


def test_parakeet_transcribe_is_readable():
    """Verify parakeet_transcribe.py is readable."""
    script_path = Path(__file__).parent.parent / "parakeet_transcribe.py"
    content = script_path.read_text()
    assert len(content) > 0, "parakeet_transcribe.py should not be empty"


def test_parakeet_transcribe_has_main():
    """Verify parakeet_transcribe.py has main execution block."""
    script_path = Path(__file__).parent.parent / "parakeet_transcribe.py"
    content = script_path.read_text()
    assert "if __name__" in content, "Script should have main execution block"


def test_parakeet_transcribe_imports():
    """Verify parakeet_transcribe.py has expected imports."""
    script_path = Path(__file__).parent.parent / "parakeet_transcribe.py"
    content = script_path.read_text()
    # Check for NeMo imports
    assert "nemo" in content.lower(), "Script should import NeMo"
