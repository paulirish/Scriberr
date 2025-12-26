"""Tests for canary_transcribe.py"""
import pytest
from pathlib import Path


def test_canary_transcribe_exists():
    """Verify canary_transcribe.py exists."""
    script_path = Path(__file__).parent.parent / "canary_transcribe.py"
    assert script_path.exists(), "canary_transcribe.py should exist"


def test_canary_transcribe_is_readable():
    """Verify canary_transcribe.py is readable."""
    script_path = Path(__file__).parent.parent / "canary_transcribe.py"
    content = script_path.read_text()
    assert len(content) > 0, "canary_transcribe.py should not be empty"


def test_canary_transcribe_has_main():
    """Verify canary_transcribe.py has main execution block."""
    script_path = Path(__file__).parent.parent / "canary_transcribe.py"
    content = script_path.read_text()
    assert "if __name__" in content, "Script should have main execution block"


def test_canary_transcribe_imports():
    """Verify canary_transcribe.py has expected imports."""
    script_path = Path(__file__).parent.parent / "canary_transcribe.py"
    content = script_path.read_text()
    # Check for NeMo imports
    assert "nemo" in content.lower(), "Script should import NeMo"
