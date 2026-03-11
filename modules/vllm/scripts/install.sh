#!/usr/bin/env bash
set -euo pipefail

if command -v sudo >/dev/null 2>&1 && [[ "${EUID:-$(id -u)}" -ne 0 ]]; then
  SUDO="sudo"
else
  SUDO=""
fi

PYTHON_BIN="${VLLM_PYTHON:-python3}"
INSTALL_METHOD="${VLLM_INSTALL_METHOD:-auto}"
BASE_INFO_PATH="${LAS_BASE_INFO_PATH:-$HOME/.localaistack/base_info.json}"
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

has_avx512() {
  local flags=""
  if [[ -f "$BASE_INFO_PATH" ]]; then
    flags="$(grep -i "flags" "$BASE_INFO_PATH" | head -n 1 | tr -d '\r')"
  fi
  if [[ -z "$flags" && -r /proc/cpuinfo ]]; then
    flags="$(grep -i "flags" /proc/cpuinfo | head -n 1 | tr -d '\r')"
  fi
  echo "$flags" | grep -qi "avx512"
}

detect_gpu_name() {
  local gpu_name=""
  if command -v nvidia-smi >/dev/null 2>&1; then
    gpu_name="$(nvidia-smi --query-gpu=name --format=csv,noheader 2>/dev/null | head -n 1 | tr -d '\r')"
  fi
  if [[ -z "$gpu_name" && -f "$BASE_INFO_PATH" ]]; then
    gpu_name="$("$PYTHON_BIN" - "$BASE_INFO_PATH" <<'PY'
import json
import re
import sys

path = sys.argv[1]
raw = open(path, "r", encoding="utf-8").read()
try:
    data = json.loads(raw)
except Exception:
    match = re.search(r"(?im)-\s*GPU(?:\s*\([^)]+\)|（[^）]+）)?\s*[:：]\s*([^\n#]+)", raw)
    if match:
        print(match.group(1).strip())
else:
    gpu = data.get("gpu") or data.get("gpu_name") or ""
    if isinstance(gpu, str):
        print(gpu.split(";")[0].strip())
PY
)"
  fi
  printf '%s' "$gpu_name"
}

is_v100_gpu() {
  local gpu_name
  gpu_name="$(detect_gpu_name)"
  [[ "$gpu_name" =~ (Tesla[[:space:]]+)?V100 ]]
}

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

delegate_to_v100_installer_if_needed() {
  if [[ "${VLLM_FORCE_GENERIC_INSTALL:-0}" == "1" ]]; then
    return
  fi
  if [[ "${VLLM_FORCE_V100_INSTALL:-0}" == "1" ]] || is_v100_gpu; then
    echo "Detected Tesla V100 GPU, delegating to install_for_v100.sh" >&2
    exec bash "$SCRIPT_DIR/install_for_v100.sh"
  fi
}

install_from_source() {
  if ! command -v git >/dev/null 2>&1; then
    echo "git is required for source installs. Install git or use VLLM_INSTALL_METHOD=wheel." >&2
    exit 1
  fi

  if ! command -v uv >/dev/null 2>&1; then
    if ! command -v curl >/dev/null 2>&1; then
      echo "curl is required to auto-install uv. Install curl or uv, or use VLLM_INSTALL_METHOD=wheel." >&2
      exit 1
    fi
    curl -LsSf https://astral.sh/uv/install.sh | sh
    export PATH="$HOME/.local/bin:$PATH"
    if ! command -v uv >/dev/null 2>&1; then
      echo "uv install finished but uv is still not in PATH. Try: export PATH=\"\$HOME/.local/bin:\$PATH\"" >&2
      exit 1
    fi
  fi

  source_dir="${VLLM_SOURCE_DIR:-$HOME/vllm}"
  repo_url="${VLLM_REPO_URL:-https://github.com/vllm-project/vllm.git}"

  if [[ ! -d "$source_dir/.git" ]]; then
    git clone "$repo_url" "$source_dir"
  fi

  pushd "$source_dir" >/dev/null

  uv venv --allow-existing --python "${VLLM_PYTHON_VERSION:-3.12}" --seed
  # shellcheck disable=SC1091
  source .venv/bin/activate
  if ! VLLM_USE_PRECOMPILED=1 uv pip install --editable .; then
    if [[ -z "${VLLM_PRECOMPILED_WHEEL_COMMIT:-}" ]]; then
      echo "source install failed while resolving the current precompiled wheel; retrying with nightly commit metadata" >&2
      VLLM_USE_PRECOMPILED=1 VLLM_PRECOMPILED_WHEEL_COMMIT=nightly uv pip install --editable .
    else
      exit 1
    fi
  fi

  ensure_wrapper "$source_dir/.venv/bin/vllm"

  popd >/dev/null
}

delegate_to_v100_installer_if_needed

install_from_wheel() {
  venv_dir="${VLLM_VENV_DIR:-$HOME/.localaistack/venv/vllm}"
  mkdir -p "$(dirname "$venv_dir")"
  "$PYTHON_BIN" -m venv "$venv_dir"
  "$venv_dir/bin/python" -m pip install --upgrade pip
  "$venv_dir/bin/python" -m pip install "$wheel_url" --extra-index-url https://download.pytorch.org/whl/cpu
  ensure_wrapper "$venv_dir/bin/vllm"
}

if [[ "$INSTALL_METHOD" == "auto" ]]; then
  if has_avx512; then
    INSTALL_METHOD="wheel"
  else
    if command -v git >/dev/null 2>&1 && command -v uv >/dev/null 2>&1; then
      INSTALL_METHOD="source"
    else
      echo "auto mode fallback to wheel: source install requires both git and uv." >&2
      INSTALL_METHOD="wheel"
    fi
  fi
fi

if [[ "$INSTALL_METHOD" == "source" ]]; then
  install_from_source
  exit 0
fi

if [[ -n "${VLLM_WHEEL_URL:-}" ]]; then
  wheel_url="$VLLM_WHEEL_URL"
else
  if [[ -z "${VLLM_VERSION:-}" ]]; then
    VLLM_VERSION="$($PYTHON_BIN - <<'PY'
import json
import urllib.request

url = "https://api.github.com/repos/vllm-project/vllm/releases/latest"
with urllib.request.urlopen(url, timeout=20) as response:
    data = json.load(response)

version = data.get("tag_name", "").lstrip("v")
print(version)
PY
)"
  fi

  if [[ -z "${VLLM_VERSION:-}" ]]; then
    echo "Failed to resolve VLLM_VERSION. Set VLLM_VERSION or VLLM_WHEEL_URL." >&2
    exit 1
  fi

  arch="$(uname -m)"
  case "$arch" in
    x86_64)
      wheel_name="vllm-${VLLM_VERSION}+cpu-cp38-abi3-manylinux_2_35_x86_64.whl"
      ;;
    aarch64)
      wheel_name="vllm-${VLLM_VERSION}+cpu-cp38-abi3-manylinux_2_35_aarch64.whl"
      ;;
    *)
      echo "Unsupported architecture for CPU wheel: $arch" >&2
      exit 1
      ;;
  esac

  wheel_url="https://github.com/vllm-project/vllm/releases/download/v${VLLM_VERSION}/${wheel_name}"
fi

install_from_wheel
