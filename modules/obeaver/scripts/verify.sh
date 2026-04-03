#!/usr/bin/env bash
set -euo pipefail

MODULE_HOME="${OBEAVER_HOME:-${HOME}/.localaistack/tools/obeaver}"
LEGACY_HOME="${HOME}/.localaistack/tools/ebeaver"
if [[ -z "${OBEAVER_HOME:-}" && ! -d "$MODULE_HOME" && -d "$LEGACY_HOME" ]]; then
  MODULE_HOME="$LEGACY_HOME"
fi
VENV_DIR="${MODULE_HOME}/.venv"
LOCAL_BIN_DIR="${HOME}/.local/bin"

find_obeaver() {
  local -a candidates=(
    "$(command -v obeaver 2>/dev/null || true)"
    "${LOCAL_BIN_DIR}/obeaver"
    "${VENV_DIR}/bin/obeaver"
  )

  local candidate=""
  for candidate in "${candidates[@]}"; do
    if [[ -n "$candidate" && -x "$candidate" ]]; then
      printf '%s\n' "$candidate"
      return 0
    fi
  done

  return 1
}

main() {
  local obeaver_bin=""

  if ! obeaver_bin="$(find_obeaver)"; then
    echo "oBeaver CLI is not available in PATH, ~/.local/bin, or the managed virtual environment." >&2
    return 1
  fi

  "$obeaver_bin" --help >/dev/null 2>&1 || {
    echo "Failed to run '$obeaver_bin --help'." >&2
    return 1
  }

  "$obeaver_bin" version >/dev/null 2>&1 || {
    echo "Failed to run '$obeaver_bin version'." >&2
    return 1
  }

  echo "oBeaver verification succeeded: ${obeaver_bin}"
}

main "$@"
