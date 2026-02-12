#!/usr/bin/env bash
set -euo pipefail

command -v openclaw >/dev/null 2>&1 || {
  echo "OpenClaw CLI is not available in PATH." >&2
  exit 1
}

openclaw --help >/dev/null 2>&1 || {
  echo "Failed to run 'openclaw --help'." >&2
  exit 1
}

echo "OpenClaw verification succeeded."
