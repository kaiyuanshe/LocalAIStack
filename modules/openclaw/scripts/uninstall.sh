#!/usr/bin/env bash
set -euo pipefail

if command -v sudo >/dev/null 2>&1 && [[ "${EUID:-$(id -u)}" -ne 0 ]]; then
  SUDO="sudo"
else
  SUDO=""
fi

if command -v openclaw >/dev/null 2>&1; then
  bin_path="$(command -v openclaw)"
  $SUDO rm -f "$bin_path"
fi

$SUDO rm -f /usr/local/bin/openclaw || true
rm -f "${HOME}/.local/bin/openclaw" || true
