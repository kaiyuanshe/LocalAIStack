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
TORCH_VERSION="${VLLM_V100_TORCH_VERSION:-2.9.1}"
TORCHVISION_VERSION="${VLLM_V100_TORCHVISION_VERSION:-0.24.1}"
TORCHAUDIO_VERSION="${VLLM_V100_TORCHAUDIO_VERSION:-2.9.1}"
TRITON_VERSION="${VLLM_V100_TRITON_VERSION:-3.5.1}"
TORCH_PACKAGES="${VLLM_V100_TORCH_PACKAGES:-torch==${TORCH_VERSION} torchvision==${TORCHVISION_VERSION} torchaudio==${TORCHAUDIO_VERSION}}"
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

detect_build_parallelism() {
  local cpu_count mem_available_kb mem_based_jobs cpu_based_jobs jobs

  if [[ -n "${VLLM_V100_MAX_JOBS:-}" ]]; then
    MAX_JOBS="$VLLM_V100_MAX_JOBS"
  else
    cpu_count="$(getconf _NPROCESSORS_ONLN 2>/dev/null || nproc 2>/dev/null || echo 4)"
    mem_available_kb="$(awk '/MemAvailable:/ {print $2; exit}' /proc/meminfo 2>/dev/null)"
    if [[ -z "$mem_available_kb" ]]; then
      mem_available_kb=$((16 * 1024 * 1024))
    fi
    mem_based_jobs=$((mem_available_kb / 4194304))
    (( mem_based_jobs < 2 )) && mem_based_jobs=2
    cpu_based_jobs=$((cpu_count / 2))
    (( cpu_based_jobs < 2 )) && cpu_based_jobs=2
    jobs="$cpu_based_jobs"
    (( mem_based_jobs < jobs )) && jobs="$mem_based_jobs"
    (( jobs > 8 )) && jobs=8
    MAX_JOBS="$jobs"
  fi

  if [[ -n "${VLLM_V100_NVCC_THREADS:-}" ]]; then
    NVCC_THREADS="$VLLM_V100_NVCC_THREADS"
  else
    NVCC_THREADS=1
  fi

  CMAKE_BUILD_PARALLEL_LEVEL="${CMAKE_BUILD_PARALLEL_LEVEL:-$MAX_JOBS}"
  export MAX_JOBS
  export CMAKE_BUILD_PARALLEL_LEVEL
  export NVCC_THREADS
}

ensure_v100_runtime_stack() {
  local installed_torch=""
  local installed_triton=""

  installed_torch="$(python - <<'PY'
try:
    import torch
    print(torch.__version__.split("+")[0])
except Exception:
    pass
PY
)"

  installed_triton="$(python - <<'PY'
try:
    import triton
    print(triton.__version__)
except Exception:
    pass
PY
)"

  if [[ "$installed_torch" != "$TORCH_VERSION" ]]; then
    uv pip install --index-url "$TORCH_INDEX_URL" --force-reinstall $TORCH_PACKAGES
  fi

  if [[ "$installed_triton" != "$TRITON_VERSION" ]]; then
    uv pip install --force-reinstall "triton==$TRITON_VERSION"
  fi
}

configure_cuda_toolchain() {
  local candidate=""
  local nvcc_bin=""
  local version=""

  for candidate in "${CUDA_HOME:-}" /usr/local/cuda /usr/local/cuda-12.9 /usr/local/cuda-12.8 /usr/local/cuda-12 /usr/local/cuda-11.8; do
    [[ -n "$candidate" ]] || continue
    nvcc_bin="$candidate/bin/nvcc"
    if [[ -x "$nvcc_bin" ]]; then
      version="$("$nvcc_bin" -V 2>/dev/null | sed -n 's/.*release \([0-9][0-9.]*\).*/\1/p' | head -n 1)"
      if [[ -n "$version" ]]; then
        export CUDA_HOME="$candidate"
        export CUDACXX="$nvcc_bin"
        export PATH="$CUDA_HOME/bin:$PATH"
        export LD_LIBRARY_PATH="$CUDA_HOME/lib64:${LD_LIBRARY_PATH:-}"
        case "$version" in
          12.*|11.[6-9]*)
            return
            ;;
        esac
      fi
    fi
  done

  echo "A CUDA toolkit with nvcc >= 11.6 is required for 1Cat-vLLM. Current CUDA_HOME=${CUDA_HOME:-unset}, /usr/bin/nvcc=$(/usr/bin/nvcc -V 2>/dev/null | sed -n 's/.*release \([0-9][0-9.]*\).*/\1/p' | head -n 1)." >&2
  echo "Set CUDA_HOME to a newer toolkit, for example: export CUDA_HOME=/usr/local/cuda-12.9" >&2
  exit 1
}

require_command git
install_uv_if_needed
detect_build_parallelism
configure_cuda_toolchain

if [[ ! -d "$SOURCE_DIR/.git" ]]; then
  git clone "$REPO_URL" "$SOURCE_DIR"
fi

pushd "$SOURCE_DIR" >/dev/null

git submodule update --init --recursive lmdeploy

uv venv --allow-existing --python "$PYTHON_VERSION" --seed
# shellcheck disable=SC1091
source .venv/bin/activate

ensure_v100_runtime_stack

export TORCH_CUDA_ARCH_LIST
export VLLM_TARGET_DEVICE="${VLLM_TARGET_DEVICE:-cuda}"
echo "Using V100 build parallelism: MAX_JOBS=$MAX_JOBS, CMAKE_BUILD_PARALLEL_LEVEL=$CMAKE_BUILD_PARALLEL_LEVEL, NVCC_THREADS=$NVCC_THREADS" >&2

python use_existing_torch.py
uv pip install -r requirements/build.txt
uv pip install -r requirements/cuda.txt
uv pip install -r requirements/common.txt
python -m pip install -e . --no-build-isolation

ensure_wrapper "$SOURCE_DIR/.venv/bin/vllm"

if ! command -v "$SOURCE_DIR/.venv/bin/vllm" >/dev/null 2>&1; then
  echo "vllm executable was not created after editable install" >&2
  exit 1
fi

python - <<'PY'
import triton
import triton.language.target_info
print("Validated runtime stack:", {"triton": triton.__version__})
PY

popd >/dev/null
