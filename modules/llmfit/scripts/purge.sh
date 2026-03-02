#!/usr/bin/env bash
set -euo pipefail

if command -v sudo >/dev/null 2>&1 && [[ "${EUID:-$(id -u)}" -ne 0 ]]; then
  SUDO="sudo"
else
  SUDO=""
fi

bash "$(dirname "$0")/uninstall.sh"

rm -rf "${HOME}/.llmfit" || true
rm -rf "${HOME}/.config/llmfit" || true
rm -rf "${HOME}/.local/share/llmfit" || true
$SUDO rm -rf "/root/.config/llmfit" || true
$SUDO rm -rf "/root/.local/share/llmfit" || true
