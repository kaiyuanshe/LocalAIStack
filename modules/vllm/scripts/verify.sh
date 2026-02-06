#!/usr/bin/env bash
set -euo pipefail

VENV_DIR="${VLLM_VENV_DIR:-$HOME/.localaistack/venv/vllm}"
VENV_PY="$VENV_DIR/bin/python"

if [[ ! -x "$VENV_PY" ]]; then
  echo "vLLM venv not found at $VENV_DIR" >&2
  exit 1
fi

"$VENV_PY" - <<'PY'
import vllm
print(vllm.__version__)
PY
