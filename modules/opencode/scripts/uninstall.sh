#!/usr/bin/env bash
set -euo pipefail

if command -v sudo >/dev/null 2>&1 && [[ "${EUID:-$(id -u)}" -ne 0 ]]; then
  SUDO="sudo"
else
  SUDO=""
fi

if command -v opencode >/dev/null 2>&1; then
  bin_path="$(command -v opencode)"
  $SUDO rm -f "$bin_path"
fi
