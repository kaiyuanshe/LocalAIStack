#!/usr/bin/env bash
set -euo pipefail

COMFYUI_HOME="${COMFYUI_HOME:-$HOME/.local/share/localaistack/comfyui}"

if [[ ! -d "$COMFYUI_HOME" ]]; then
  echo "ComfyUI home not found: $COMFYUI_HOME" >&2
  exit 1
fi

if [[ ! -x "$COMFYUI_HOME/.venv/bin/python" ]]; then
  echo "ComfyUI virtualenv not found: $COMFYUI_HOME/.venv" >&2
  exit 1
fi

"$COMFYUI_HOME/.venv/bin/python" - <<'PY'
import importlib.util
if importlib.util.find_spec("torch") is None:
    raise SystemExit("torch is not installed in ComfyUI environment")
print("ok")
PY

if [[ ! -x "$HOME/.local/bin/comfyui-las" ]]; then
  echo "launcher not found: $HOME/.local/bin/comfyui-las" >&2
  exit 1
fi

echo "ComfyUI verify passed"
