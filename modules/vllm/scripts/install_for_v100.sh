#!/usr/bin/env bash
set -euo pipefail

if command -v sudo >/dev/null 2>&1 && [[ "${EUID:-$(id -u)}" -ne 0 ]]; then
  SUDO="sudo"
else
  SUDO=""
fi

PYTHON_BIN="${VLLM_PYTHON:-python3}"
SOURCE_DIR="${VLLM_V100_SOURCE_DIR:-$HOME/1Cat-vLLM}"
REPO_URL="${VLLM_V100_REPO_URL:-https://github.com/1CatAI/1Cat-vLLM.git}"
PYTHON_VERSION="${VLLM_PYTHON_VERSION:-3.12}"
TORCH_INDEX_URL="${VLLM_V100_TORCH_INDEX_URL:-https://download.pytorch.org/whl/cu128}"
TORCH_PACKAGES="${VLLM_V100_TORCH_PACKAGES:-torch==2.8.0 torchvision==0.23.0 torchaudio==2.8.0}"
TORCH_CUDA_ARCH_LIST="${TORCH_CUDA_ARCH_LIST:-7.0}"

ensure_wrapper() {
  local venv_bin="$1"
  if [[ ! -x "$venv_bin" ]]; then
    echo "vllm entrypoint not found at $venv_bin" >&2
    exit 1
  fi
  wrapper_content="#!/usr/bin/env bash
# LocalAIStack vllm wrapper
export PYTHONNOUSERSITE=1
exec \"$venv_bin\" \"\$@\"
"
  if [[ -n "$SUDO" && -d /usr/local/bin ]]; then
    $SUDO install -d /usr/local/bin
    echo "$wrapper_content" | $SUDO tee /usr/local/bin/vllm >/dev/null
    $SUDO chmod 0755 /usr/local/bin/vllm
  else
    mkdir -p "$HOME/.local/bin"
    echo "$wrapper_content" > "$HOME/.local/bin/vllm"
    chmod 0755 "$HOME/.local/bin/vllm"
  fi
}

require_command() {
  local command_name="$1"
  if ! command -v "$command_name" >/dev/null 2>&1; then
    echo "$command_name is required for V100 vLLM install" >&2
    exit 1
  fi
}

install_uv_if_needed() {
  if command -v uv >/dev/null 2>&1; then
    return
  fi
  require_command curl
  curl -LsSf https://astral.sh/uv/install.sh | sh
  export PATH="$HOME/.local/bin:$PATH"
  if ! command -v uv >/dev/null 2>&1; then
    echo "uv install finished but uv is still not in PATH. Try: export PATH=\"\$HOME/.local/bin:\$PATH\"" >&2
    exit 1
  fi
}

require_command git
install_uv_if_needed

if [[ ! -d "$SOURCE_DIR/.git" ]]; then
  git clone "$REPO_URL" "$SOURCE_DIR"
fi

pushd "$SOURCE_DIR" >/dev/null

uv venv --allow-existing --python "$PYTHON_VERSION" --seed
# shellcheck disable=SC1091
source .venv/bin/activate

if ! python -c 'import torch' >/dev/null 2>&1; then
  uv pip install --index-url "$TORCH_INDEX_URL" $TORCH_PACKAGES
fi

export TORCH_CUDA_ARCH_LIST
export VLLM_TARGET_DEVICE="${VLLM_TARGET_DEVICE:-cuda}"
python use_existing_torch.py

ensure_wrapper "$SOURCE_DIR/.venv/bin/vllm"

popd >/dev/null
