#!/usr/bin/env bash
set -euo pipefail

if command -v sudo >/dev/null 2>&1 && [[ "${EUID:-$(id -u)}" -ne 0 ]]; then
  SUDO="sudo"
else
  SUDO=""
fi

VENV_DIR="${VLLM_VENV_DIR:-$HOME/.localaistack/venv/vllm}"
VLLM_SOURCE_DIR="${VLLM_SOURCE_DIR:-$HOME/.localaistack/src/vllm}"

remove_wrapper() {
  local path="$1"
  if [[ -f "$path" ]]; then
    if grep -q "LocalAIStack vllm wrapper" "$path"; then
      if [[ -n "$SUDO" && "$path" == /usr/local/bin/* ]]; then
        $SUDO rm -f "$path"
      else
        rm -f "$path"
      fi
    fi
  fi
}

remove_wrapper "/usr/local/bin/vllm"
remove_wrapper "$HOME/.local/bin/vllm"

if [[ -d "$VENV_DIR" ]]; then
  rm -rf "$VENV_DIR"
fi

if [[ -d "$VLLM_SOURCE_DIR" ]]; then
  rm -rf "$VLLM_SOURCE_DIR"
fi
