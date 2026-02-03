#!/usr/bin/env bash
set -euo pipefail

PYTHON_BIN="${VLLM_PYTHON:-python3}"

"$PYTHON_BIN" - <<'PY'
import vllm
print(vllm.__version__)
PY
