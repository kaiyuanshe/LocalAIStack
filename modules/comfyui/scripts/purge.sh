#!/usr/bin/env bash
set -euo pipefail

COMFYUI_HOME="${COMFYUI_HOME:-$HOME/.local/share/localaistack/comfyui}"

rm -f "$HOME/.local/bin/comfyui-las"
rm -rf "$COMFYUI_HOME"

echo "ComfyUI purged from: $COMFYUI_HOME"
