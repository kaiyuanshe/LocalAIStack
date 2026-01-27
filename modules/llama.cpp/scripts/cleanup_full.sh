#!/usr/bin/env bash
set -euo pipefail

if command -v sudo >/dev/null 2>&1 && [[ "${EUID:-$(id -u)}" -ne 0 ]]; then
  SUDO="sudo"
else
  SUDO=""
fi

$SUDO rm -f /usr/local/bin/llama-cli /usr/local/bin/llama-server
$SUDO rm -rf /usr/local/llama.cpp
