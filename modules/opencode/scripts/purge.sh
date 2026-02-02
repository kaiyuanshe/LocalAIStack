#!/usr/bin/env bash
set -euo pipefail

if command -v sudo >/dev/null 2>&1 && [[ "${EUID:-$(id -u)}" -ne 0 ]]; then
  SUDO="sudo"
else
  SUDO=""
fi

bash "$(dirname "$0")/uninstall.sh"

$SUDO rm -rf "${HOME}/.opencode" || true
