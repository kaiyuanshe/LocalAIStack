#!/usr/bin/env bash
set -euo pipefail

MODULE_HOME="${OBEAVER_HOME:-${HOME}/.localaistack/tools/obeaver}"
LEGACY_HOME="${HOME}/.localaistack/tools/ebeaver"
if [[ -z "${OBEAVER_HOME:-}" && ! -d "$MODULE_HOME" && -d "$LEGACY_HOME" ]]; then
  MODULE_HOME="$LEGACY_HOME"
fi
LOCAL_BIN_DIR="${HOME}/.local/bin"
WRAPPER_PATH="${LOCAL_BIN_DIR}/obeaver"

rm -rf "${MODULE_HOME}/.venv"
rm -f "${WRAPPER_PATH}"

echo "Removed oBeaver virtual environment and launcher."
