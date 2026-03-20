#!/usr/bin/env bash
set -euo pipefail

python3 - <<'PY'
import sys
if sys.version_info >= (3, 14):
    raise SystemExit("Unsloth supports Python 3.13 or lower")
PY

python3 -m pip install --upgrade --user "unsloth"
