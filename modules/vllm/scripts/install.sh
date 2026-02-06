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
}

fetch_versions_list() {
  "$PYTHON_BIN" - <<'PY'
import json
import urllib.request

url = "https://api.github.com/repos/vllm-project/vllm/releases?per_page=20"
with urllib.request.urlopen(url, timeout=20) as response:
    data = json.load(response)
for release in data:
    if release.get("prerelease"):
        continue
    tag = release.get("tag_name", "")
    if not tag:
        continue
    print(tag.lstrip("v"))
PY
}

resolve_wheel_url() {
  local version="$1"
  local arch
  arch="$(uname -m)"
  case "$arch" in
    x86_64)
      echo "https://github.com/vllm-project/vllm/releases/download/v${version}/vllm-${version}+cpu-cp38-abi3-manylinux_2_35_x86_64.whl"
      ;;
    aarch64)
      echo "https://github.com/vllm-project/vllm/releases/download/v${version}/vllm-${version}+cpu-cp38-abi3-manylinux_2_35_aarch64.whl"
      ;;
    *)
      echo ""
      ;;
  esac
}

ensure_build_deps() {
  if command -v apt-get >/dev/null 2>&1 && [[ -n "$SUDO" ]]; then
    $SUDO apt-get update
    $SUDO apt-get install -y build-essential git cmake ninja-build python3-dev
  fi
}

ensure_git_repo() {
  if ! command -v git >/dev/null 2>&1; then
    echo "git is required to install vLLM from source." >&2
    exit 1
  fi
  if [[ -d "$VLLM_SOURCE_DIR/.git" ]]; then
    git -C "$VLLM_SOURCE_DIR" fetch --all --tags --prune
    git -C "$VLLM_SOURCE_DIR" pull --ff-only
  else
    mkdir -p "$(dirname "$VLLM_SOURCE_DIR")"
    git clone https://github.com/vllm-project/vllm.git "$VLLM_SOURCE_DIR"
  fi
}

ensure_venv_with_uv() {
  if ! command -v uv >/dev/null 2>&1; then
    echo "uv is required for source install. Please install uv and retry." >&2
    exit 1
  fi
  uv venv --python "$VLLM_UV_PYTHON" --seed "$VENV_DIR"
}

install_from_source_uv() {
  echo "Falling back to source install (uv + editable) ..."
  ensure_git_repo
  ensure_build_deps

  rm -rf "$VENV_DIR"
  ensure_venv_with_uv

  VENV_PY="$VENV_DIR/bin/python"
  VENV_BIN="$VENV_DIR/bin/vllm"

  if ! "$VENV_PY" -m pip --version >/dev/null 2>&1; then
    "$VENV_PY" -m ensurepip --upgrade >/dev/null 2>&1 || true
  fi

  VLLM_USE_PRECOMPILED=1 uv pip install --editable "$VLLM_SOURCE_DIR" --python "$VENV_PY" >/tmp/las_vllm_build.log 2>&1
  if ! PYTHONNOUSERSITE=1 "$VENV_BIN" --help >/tmp/las_vllm_smoke.log 2>&1; then
    echo "Source install completed but vllm failed to run. Check /tmp/las_vllm_smoke.log" >&2
    exit 1
  fi
}

if [[ -n "${VLLM_WHEEL_URL:-}" ]]; then
  wheel_url="$VLLM_WHEEL_URL"
else
  if [[ -z "${VLLM_VERSION:-}" ]]; then
    if has_avx512; then
      VLLM_VERSION="$(fetch_latest_version)"
    else
      VLLM_VERSION=""
    fi
  fi

  if [[ -z "${VLLM_VERSION:-}" ]]; then
    VLLM_VERSION=""
  fi
fi

if [[ ! -d "$VENV_DIR" ]]; then
  if ! "$PYTHON_BIN" -m venv "$VENV_DIR" 2>/tmp/las_vllm_venv.err; then
    if grep -qi "ensurepip" /tmp/las_vllm_venv.err; then
      if command -v apt-get >/dev/null 2>&1 && [[ -n "$SUDO" ]]; then
        $SUDO apt-get update
        $SUDO apt-get install -y python3-venv
        "$PYTHON_BIN" -m venv "$VENV_DIR"
      else
        echo "Failed to create venv: ensurepip is unavailable. Install python3-venv and retry." >&2
        cat /tmp/las_vllm_venv.err >&2
        exit 1
      fi
    else
      echo "Failed to create venv at $VENV_DIR" >&2
      cat /tmp/las_vllm_venv.err >&2
      exit 1
    fi
  fi
fi

VENV_PY="$VENV_DIR/bin/python"
VENV_BIN="$VENV_DIR/bin/vllm"

if ! "$VENV_PY" -m pip --version >/dev/null 2>&1; then
  if ! "$VENV_PY" -m ensurepip --upgrade >/dev/null 2>&1; then
    if command -v apt-get >/dev/null 2>&1 && [[ -n "$SUDO" ]]; then
      $SUDO apt-get update
      $SUDO apt-get install -y python3-venv python3-pip
      "$VENV_PY" -m ensurepip --upgrade
    else
      echo "pip is missing in venv. Install python3-venv/python3-pip and retry." >&2
      exit 1
    fi
  fi
fi

"$VENV_PY" -m pip install --upgrade pip

install_and_test() {
  local version="$1"
  local url="$2"
  if [[ -z "$url" ]]; then
    return 1
  fi
  "$VENV_PY" -m pip install --upgrade --force-reinstall "$url" --extra-index-url https://download.pytorch.org/whl/cpu >/tmp/las_vllm_install.log 2>&1 || return 1
  if ! "$VENV_PY" - <<'PY'
import vllm
print(vllm.__version__)
PY
  then
    return 1
  fi
  if ! PYTHONNOUSERSITE=1 "$VENV_BIN" --help >/tmp/las_vllm_smoke.log 2>&1; then
    return 1
  fi
}

if [[ -z "${wheel_url:-}" ]]; then
  if [[ -n "${VLLM_VERSION:-}" ]]; then
    wheel_url="$(resolve_wheel_url "$VLLM_VERSION")"
    if ! install_and_test "$VLLM_VERSION" "$wheel_url"; then
      echo "Failed to install vLLM ${VLLM_VERSION}." >&2
      exit 1
    fi
  else
    versions="$(fetch_versions_list)"
    for version in $versions; do
      wheel_url="$(resolve_wheel_url "$version")"
      if install_and_test "$version" "$wheel_url"; then
        VLLM_VERSION="$version"
        break
      fi
      "$VENV_PY" -m pip uninstall -y vllm >/dev/null 2>&1 || true
    done
    if [[ -z "${VLLM_VERSION:-}" ]]; then
      install_from_source_uv
    fi
  fi
else
  "$VENV_PY" -m pip install "$wheel_url" --extra-index-url https://download.pytorch.org/whl/cpu
fi

wrapper_content="#!/usr/bin/env bash
# LocalAIStack vllm wrapper
export PYTHONNOUSERSITE=1
exec \"$VENV_BIN\" \"\$@\"
"

target_bin=""
if [[ -n "$SUDO" && -d /usr/local/bin ]]; then
  target_bin="/usr/local/bin/vllm"
  $SUDO install -d /usr/local/bin
  echo "$wrapper_content" | $SUDO tee "$target_bin" >/dev/null
  $SUDO chmod 0755 "$target_bin"
else
  mkdir -p "$HOME/.local/bin"
  target_bin="$HOME/.local/bin/vllm"
  echo "$wrapper_content" > "$target_bin"
  chmod 0755 "$target_bin"
fi

echo "vllm installed with venv at $VENV_DIR"
echo "wrapper: $target_bin"
