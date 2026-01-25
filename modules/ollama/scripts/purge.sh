#!/usr/bin/env bash
set -euo pipefail

bash "$(dirname "$0")/uninstall.sh"

rm -rf "${HOME}/.ollama"
echo "Ollama fully removed (including data)."
