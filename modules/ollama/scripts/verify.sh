#!/usr/bin/env bash
set -euo pipefail

# Service endpoint should respond
api_response="$(curl --fail --silent --show-error --max-time 5 http://127.0.0.1:11434/api/tags)" || {
  echo "Failed to reach Ollama API at http://127.0.0.1:11434/api/tags." >&2
  exit 1
}
if ! grep -q '"models"' <<<"${api_response}"; then
  echo "Ollama API response did not include models: ${api_response}" >&2
  exit 1
fi
echo "Ollama API endpoint is responding."

# CLI should be present
command -v ollama >/dev/null || {
  echo "Ollama CLI is not available in PATH." >&2
  exit 1
}
ollama --version >/dev/null || {
  echo "Failed to run 'ollama --version'." >&2
  exit 1
}
echo "Ollama CLI is available."

echo "Ollama verification succeeded."
