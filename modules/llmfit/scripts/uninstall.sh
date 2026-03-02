#!/usr/bin/env bash
set -euo pipefail

if command -v sudo >/dev/null 2>&1 && [[ "${EUID:-$(id -u)}" -ne 0 ]]; then
  SUDO="sudo"
else
  SUDO=""
fi

if command -v llmfit >/dev/null 2>&1; then
  bin_path="$(command -v llmfit)"
  $SUDO rm -f "$bin_path"
fi

$SUDO rm -f /usr/local/bin/llmfit || true
rm -f "${HOME}/.local/bin/llmfit" || true
