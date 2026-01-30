#!/usr/bin/env bash
set -euo pipefail

python3 - <<'PY'
import importlib.util
spec = importlib.util.find_spec("huggingface_hub")
if spec is None:
    raise SystemExit("huggingface_hub is not installed")
import huggingface_hub
print(huggingface_hub.__version__)
PY

python3 - <<'PY'
import importlib.metadata as md

eps = md.entry_points()
group = eps.select(group="console_scripts") if hasattr(eps, "select") else eps.get("console_scripts", [])
names = {ep.name for ep in group}
if not (names & {"hf", "huggingface-cli"}):
    raise SystemExit(f"Expected hf or huggingface-cli console script, found: {sorted(names)}")
PY
