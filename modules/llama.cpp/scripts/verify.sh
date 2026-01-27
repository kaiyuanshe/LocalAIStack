#!/usr/bin/env bash
set -euo pipefail

command -v llama-cli >/dev/null 2>&1
command -v llama-server >/dev/null 2>&1 || true
