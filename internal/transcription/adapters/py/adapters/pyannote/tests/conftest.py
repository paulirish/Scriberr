"""Pytest configuration for PyAnnote adapter tests."""
import sys
from pathlib import Path

# Add parent directory to path so we can import scripts if needed
sys.path.insert(0, str(Path(__file__).parent.parent))
