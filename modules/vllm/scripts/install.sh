#!/usr/bin/env bash
set -euo pipefail

if command -v sudo >/dev/null 2>&1 && [[ "${EUID:-$(id -u)}" -ne 0 ]]; then
  SUDO="sudo"
else
  SUDO=""
fi

PYTHON_BIN="${VLLM_PYTHON:-python3}"
INSTALL_METHOD="${VLLM_INSTALL_METHOD:-wheel}"

install_from_source() {
  if ! command -v git >/dev/null 2>&1; then
    echo "git is required for source installs. Install git or use VLLM_INSTALL_METHOD=wheel." >&2
    exit 1
  fi

  if ! command -v uv >/dev/null 2>&1; then
    echo "uv is required for source installs. Install uv or use VLLM_INSTALL_METHOD=wheel." >&2
    exit 1
  fi

  source_dir="${VLLM_SOURCE_DIR:-$HOME/vllm}"
  repo_url="${VLLM_REPO_URL:-https://github.com/vllm-project/vllm.git}"

  if [[ ! -d "$source_dir/.git" ]]; then
    git clone "$repo_url" "$source_dir"
  fi

  pushd "$source_dir" >/dev/null

  uv venv --python "${VLLM_PYTHON_VERSION:-3.12}" --seed
  # shellcheck disable=SC1091
  source .venv/bin/activate
  VLLM_USE_PRECOMPILED=1 uv pip install --editable .

  popd >/dev/null
}

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

$SUDO "$PYTHON_BIN" -m pip install --upgrade pip
$SUDO "$PYTHON_BIN" -m pip install "$wheel_url" --extra-index-url https://download.pytorch.org/whl/cpu
