#!/usr/bin/env bash
set -euo pipefail

# Service endpoint should respond
curl -s http://127.0.0.1:11434/api/tags | grep -q '"models"'
echo "Ollama API endpoint is responding."

# CLI should be present
command -v ollama >/dev/null
ollama --version >/dev/null
echo "Ollama CLI is available."

echo "Ollama verification succeeded."
