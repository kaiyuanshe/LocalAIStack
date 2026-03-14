#!/usr/bin/env bash
set -euo pipefail

NVM_DIR="${NVM_DIR:-${HOME}/.nvm}"

load_nvm() {
  if [[ -s "${NVM_DIR}/nvm.sh" ]]; then
    # shellcheck disable=SC1090
    source "${NVM_DIR}/nvm.sh"
    return 0
  fi

  return 1
}

find_openclaw_bin() {
  local candidate=""
  local -a candidates=()

  if candidate="$(command -v openclaw 2>/dev/null || true)" && [[ -n "$candidate" ]]; then
    printf '%s\n' "$candidate"
    return 0
  fi

  candidates+=(
    "${HOME}/.local/bin/openclaw"
    "/usr/local/bin/openclaw"
    "/root/.local/bin/openclaw"
  )

  if load_nvm; then
    if candidate="$(command -v openclaw 2>/dev/null || true)" && [[ -n "$candidate" ]]; then
      printf '%s\n' "$candidate"
      return 0
    fi

    if [[ "$(nvm version default 2>/dev/null || true)" != "N/A" ]]; then
      nvm use default >/dev/null 2>&1 || true
      if candidate="$(command -v openclaw 2>/dev/null || true)" && [[ -n "$candidate" ]]; then
        printf '%s\n' "$candidate"
        return 0
      fi
    fi
  fi

  if [[ -d "${NVM_DIR}/versions/node" ]]; then
    while IFS= read -r candidate; do
      candidates+=("$candidate")
    done < <(find "${NVM_DIR}/versions/node" -maxdepth 4 -type f -path '*/bin/openclaw' 2>/dev/null)
  fi

  for candidate in "${candidates[@]}"; do
    if [[ -x "$candidate" ]]; then
      printf '%s\n' "$candidate"
      return 0
    fi
  done

  return 1
}

main() {
  local openclaw_bin=""

  if ! openclaw_bin="$(find_openclaw_bin)"; then
    echo "OpenClaw CLI is not available in PATH, common install locations, or NVM-managed Node bins." >&2
    return 1
  fi

  "$openclaw_bin" --help >/dev/null 2>&1 || {
    echo "Failed to run '$openclaw_bin --help'." >&2
    return 1
  }

  echo "OpenClaw verification succeeded: $openclaw_bin"
}

main "$@"
