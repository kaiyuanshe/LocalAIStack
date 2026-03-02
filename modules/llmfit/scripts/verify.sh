#!/usr/bin/env bash
set -euo pipefail

command -v llmfit >/dev/null 2>&1 || {
  echo "llmfit CLI is not available in PATH." >&2
  exit 1
}

llmfit --help >/dev/null 2>&1 || {
  echo "Failed to run 'llmfit --help'." >&2
  exit 1
}

echo "llmfit verification succeeded."
