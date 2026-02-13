#!/usr/bin/env bash
set -euo pipefail

if [[ $# -lt 1 ]]; then
  echo "Usage: setting.sh <model-bundle>" >&2
  echo "Example: setting.sh Comfy-Org_z_image_turbo" >&2
  exit 1
fi

MODEL_BUNDLE="$1"
COMFYUI_HOME="${COMFYUI_HOME:-$HOME/.local/share/localaistack/comfyui}"
SRC_BASE="${MODELS_HOME:-$HOME/.localaistack/models}"
SRC="$SRC_BASE/$MODEL_BUNDLE/split_files"

if [[ ! -d "$SRC" ]]; then
  echo "Model split_files directory not found: $SRC" >&2
  exit 1
fi

mkdir -p "$COMFYUI_HOME/models/diffusion_models" "$COMFYUI_HOME/models/text_encoders" "$COMFYUI_HOME/models/vae"

case "$MODEL_BUNDLE" in
  Comfy-Org_z_image_turbo)
    ln -sfn "$SRC/diffusion_models/z_image_turbo_bf16.safetensors" "$COMFYUI_HOME/models/diffusion_models/z_image_turbo_bf16.safetensors"
    ln -sfn "$SRC/diffusion_models/z_image_turbo_nvfp4.safetensors" "$COMFYUI_HOME/models/diffusion_models/z_image_turbo_nvfp4.safetensors"

    ln -sfn "$SRC/text_encoders/qwen_3_4b.safetensors" "$COMFYUI_HOME/models/text_encoders/qwen_3_4b.safetensors"
    ln -sfn "$SRC/text_encoders/qwen_3_4b_fp4_mixed.safetensors" "$COMFYUI_HOME/models/text_encoders/qwen_3_4b_fp4_mixed.safetensors"
    ln -sfn "$SRC/text_encoders/qwen_3_4b_fp8_mixed.safetensors" "$COMFYUI_HOME/models/text_encoders/qwen_3_4b_fp8_mixed.safetensors"

    ln -sfn "$SRC/vae/ae.safetensors" "$COMFYUI_HOME/models/vae/ae.safetensors"
    ;;
  *)
    echo "Unsupported model bundle: $MODEL_BUNDLE" >&2
    echo "Currently supported: Comfy-Org_z_image_turbo" >&2
    exit 1
    ;;
esac

echo "ComfyUI model links configured for: $MODEL_BUNDLE"
echo "ComfyUI home: $COMFYUI_HOME"
