#!/usr/bin/env bash
set -euo pipefail

python3 - <<'PY'
import importlib.metadata as md
import importlib.util

spec = importlib.util.find_spec("unsloth")
if spec is None:
    raise SystemExit("unsloth is not installed")

print(md.version("unsloth"))
PY
