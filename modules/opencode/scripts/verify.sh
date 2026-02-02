#!/usr/bin/env bash
set -euo pipefail

command -v opencode >/dev/null 2>&1 || {
  echo "OpenCode CLI is not available in PATH." >&2
  exit 1
}

opencode --help >/dev/null 2>&1 || {
  echo "Failed to run 'opencode --help'." >&2
  exit 1
}

echo "OpenCode verification succeeded."
