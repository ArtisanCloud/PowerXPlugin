#!/usr/bin/env python3
import argparse
import subprocess
import sys

parser = argparse.ArgumentParser(description="Run command with optional timeout and error reporting.")
parser.add_argument("command", nargs=argparse.REMAINDER, help="Command to execute")
parser.add_argument("--timeout", type=int, default=0, help="Timeout in seconds (0 = no timeout)")
args = parser.parse_args()

if not args.command:
    print("No command provided", file=sys.stderr)
    sys.exit(2)

try:
    subprocess.run(args.command, check=True, timeout=args.timeout or None)
except subprocess.TimeoutExpired:
    print(f"Command exceeded {args.timeout}s timeout", file=sys.stderr)
    sys.exit(124)
except subprocess.CalledProcessError as exc:
    print(f"Command failed with exit code {exc.returncode}", file=sys.stderr)
    sys.exit(exc.returncode)
