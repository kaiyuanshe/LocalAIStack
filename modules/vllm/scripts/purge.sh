#!/usr/bin/env bash
set -euo pipefail

if command -v sudo >/dev/null 2>&1 && [[ "${EUID:-$(id -u)}" -ne 0 ]]; then
  SUDO="sudo"
else
  SUDO=""
fi

VENV_DIR="${VLLM_VENV_DIR:-$HOME/.localaistack/venv/vllm}"
VLLM_SOURCE_DIR="${VLLM_SOURCE_DIR:-$HOME/vllm}"
ALT_VENV_DIRS=(
  "$HOME/vllm/.venv"
  "$HOME/.venv/vllm"
  "$HOME/.localaistack/venv/vllm"
)
CACHE_DIRS=(
  "$HOME/.cache/vllm"
  "$HOME/.config/vllm"
)

stop_vllm_processes() {
  pkill -f "/bin/vllm" 2>/dev/null || true
  pkill -f "vllm serve" 2>/dev/null || true
  pkill -f "vllm.entrypoints" 2>/dev/null || true
}

remove_wrapper() {
  local path="$1"
  if [[ -f "$path" ]]; then
    if grep -q "LocalAIStack vllm wrapper" "$path" || grep -q "\.venv/bin/vllm" "$path"; then
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

remove_venv_dir() {
  local dir="$1"
  if [[ -n "$dir" && -d "$dir" ]]; then
    rm -rf "$dir"
  fi
}

detect_and_remove_linked_venv() {
  local bin_path
  bin_path="$(command -v vllm 2>/dev/null || true)"
  if [[ -n "$bin_path" && -f "$bin_path" ]]; then
    local resolved
    resolved="$(readlink -f "$bin_path" 2>/dev/null || true)"
    if [[ "$resolved" == */.venv/bin/vllm ]]; then
      remove_venv_dir "$(dirname "$(dirname "$resolved")")"
    fi
  fi
}

detect_and_remove_linked_venv
stop_vllm_processes
remove_venv_dir "$VENV_DIR"
for dir in "${ALT_VENV_DIRS[@]}"; do
  remove_venv_dir "$dir"
done

if [[ -d "$VLLM_SOURCE_DIR" ]]; then
  rm -rf "$VLLM_SOURCE_DIR"
fi
if [[ -d "$HOME/.localaistack/src/vllm" ]]; then
  rm -rf "$HOME/.localaistack/src/vllm"
fi
for dir in "${CACHE_DIRS[@]}"; do
  remove_venv_dir "$dir"
done
