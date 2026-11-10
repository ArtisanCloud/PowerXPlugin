#!/usr/bin/env python3
"""Build the matching AI Media model-limit SDK, Nuxt and CLI development bundle."""
import subprocess
import sys
from pathlib import Path
root = Path(__file__).resolve().parents[2]
subprocess.run([sys.executable, str(root / "scripts/contracts/package-media-thumbnail64.py"), "--profile", "media-limits"], cwd=root, check=True)
