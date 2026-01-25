#!/usr/bin/env bash
set -euo pipefail

# Service endpoint should respond
curl -s http://127.0.0.1:11434/api/tags | grep -q '"models"'

# CLI should be present
command -v ollama >/dev/null
ollama --version >/dev/null
