#!/usr/bin/env bash
set -euo pipefail

mode="${1:-}"
if [[ -z "$mode" ]]; then
  echo "Usage: $0 <binary|source>" >&2
  exit 1
fi

if [[ "$mode" != "binary" && "$mode" != "source" ]]; then
  echo "Unknown mode: $mode" >&2
  exit 1
fi

if command -v sudo >/dev/null 2>&1 && [[ "${EUID:-$(id -u)}" -ne 0 ]]; then
  SUDO="sudo"
else
  SUDO=""
fi

is_truthy() {
  local value
  value="$(echo "${1:-}" | tr '[:upper:]' '[:lower:]')"
  [[ "$value" == "1" || "$value" == "on" || "$value" == "true" || "$value" == "yes" ]]
}

is_falsy() {
  local value
  value="$(echo "${1:-}" | tr '[:upper:]' '[:lower:]')"
  [[ "$value" == "0" || "$value" == "off" || "$value" == "false" || "$value" == "no" ]]
}

has_nvidia_gpu() {
  command -v nvidia-smi >/dev/null 2>&1 && nvidia-smi -L >/dev/null 2>&1
}

cuda_requested=0
if [[ -n "${LLAMA_CUDA:-}" ]]; then
  if is_truthy "${LLAMA_CUDA}"; then
    cuda_requested=1
  elif is_falsy "${LLAMA_CUDA}"; then
    cuda_requested=0
  fi
elif [[ "$mode" == "source" ]] && has_nvidia_gpu; then
  cuda_requested=1
fi

base_packages=(curl ca-certificates tar python3)
build_packages=()
cuda_host_packages=()
webui_packages=()  # populated for source mode; kept empty for binary mode (set -u safety)

# ---------------------------------------------------------------------------
# ensure_node_for_webui
#
# The llama.cpp cmake build runs with sudo, but ui-assets.cmake invokes
# npm.  If Node.js is installed via nvm (user-level, in ~/.nvm), sudo
# cannot discover it.  This function guarantees a usable node + npm for
# root by:
#   1. Checking whether root already has a suitable node (≥ 18).
#   2. If not, trying nvm: source ~/.nvm/nvm.sh, install LTS, symlink
#      the resulting binaries to /usr/local/bin.
#   3. Otherwise returning non-zero so the caller installs via the
#      system package manager (apt / dnf / …).
#
# Returns 0 when node and npm are usable by root after the call;
# non-zero when a system-package fallback is needed.
# ---------------------------------------------------------------------------
ensure_node_for_webui() {
  local min_version=18

  # 1. Already available for root?
  if $SUDO command -v node >/dev/null 2>&1 && $SUDO command -v npm >/dev/null 2>&1; then
    local node_ver
    node_ver=$($SUDO node -v 2>/dev/null | sed 's/^v//' | cut -d. -f1)
    if [[ "$node_ver" =~ ^[0-9]+$ && "$node_ver" -ge "$min_version" ]]; then
      echo "Node.js v$($SUDO node -v) already available for root."
      return 0
    fi
    echo "Node.js found but version $node_ver < $min_version; will upgrade."
  fi

  # 2. Try nvm (user-level; discovered via $HOME)
  #    nvm.sh is not `set -u` clean, so we relax nounset for the
  #    source + nvm commands and restore it afterwards.
  local nvm_script="${HOME}/.nvm/nvm.sh"
  if [[ -s "$nvm_script" ]]; then
    echo "Found nvm at ${nvm_script}; installing Node.js LTS …"

    set +u
    # shellcheck disable=SC1090
    source "$nvm_script" || { echo "Failed to source nvm.sh" >&2; set -u; return 1; }
    nvm install --lts || true   # "already installed" is non-fatal
    if ! nvm use --lts; then
      echo "nvm use --lts failed" >&2
      set -u
      return 1
    fi
    set -u

    local node_bin npm_bin
    node_bin="$(command -v node 2>/dev/null || true)"
    npm_bin="$(command -v npm 2>/dev/null || true)"

    if [[ -z "$node_bin" || -z "$npm_bin" ]]; then
      echo "nvm reported success but node/npm are not on PATH." >&2
      return 1
    fi

    $SUDO ln -sf "$node_bin" /usr/local/bin/node
    $SUDO ln -sf "$npm_bin"  /usr/local/bin/npm
    echo "Node.js $(node -v) installed via nvm → /usr/local/bin/node"
    return 0
  fi

  # 3. Neither nvm nor a suitable node for root — caller will use system packages
  echo "nvm not found; will install Node.js via system package manager."
  return 1
}

if [[ "$mode" == "source" ]]; then
  build_packages=(git cmake make gcc g++)
  # Node.js and npm are needed to build the embedded Web UI (tools/ui).
  # Prefer nvm (user-level, modern) over system packages, then symlink so
  # the sudo cmake build can find node/npm.
  if ensure_node_for_webui; then
    webui_packages=()
  else
    webui_packages=(nodejs npm)
  fi
fi

if [[ "$cuda_requested" -eq 1 ]]; then
  if [[ "$mode" == "source" ]]; then
    cuda_host_packages=(gcc-10 g++-10)
  fi
fi

install_with_apt() {
  local cuda_packages=()

  if ! $SUDO apt-get update -y; then
    echo "apt-get update failed; retrying with only the default sources.list (ignoring sources.list.d entries)." >&2
    $SUDO apt-get update -y -o Dir::Etc::sourceparts="-"
  fi

  if [[ "${#cuda_host_packages[@]}" -gt 0 ]]; then
    if apt-cache show gcc-10 >/dev/null 2>&1 && apt-cache show g++-10 >/dev/null 2>&1; then
      cuda_packages=(gcc-10 g++-10)
    elif apt-cache show gcc-11 >/dev/null 2>&1 && apt-cache show g++-11 >/dev/null 2>&1; then
      cuda_packages=(gcc-11 g++-11)
    else
      # Newer distros (e.g. Ubuntu 24.04) may not ship gcc-10/g++-10 in default repos.
      cuda_packages=(gcc g++)
    fi
  fi

  $SUDO apt-get install -y "${base_packages[@]}" "${build_packages[@]}" "${webui_packages[@]}" "${cuda_packages[@]}"
}

install_with_dnf() {
  $SUDO dnf install -y "${base_packages[@]}" "${build_packages[@]/g++/gcc-c++}" "${webui_packages[@]}"
}

install_with_yum() {
  $SUDO yum install -y "${base_packages[@]}" "${build_packages[@]/g++/gcc-c++}" "${webui_packages[@]}"
}

install_with_pacman() {
  $SUDO pacman -Sy --noconfirm "${base_packages[@]}" "${build_packages[@]}" "${webui_packages[@]}"
}

install_with_zypper() {
  $SUDO zypper --non-interactive install "${base_packages[@]}" "${build_packages[@]/g++/gcc-c++}" "${webui_packages[@]}"
}

install_with_apk() {
  $SUDO apk add --no-cache "${base_packages[@]}" "${build_packages[@]}" "${webui_packages[@]}"
}

if command -v apt-get >/dev/null 2>&1; then
  install_with_apt
elif command -v dnf >/dev/null 2>&1; then
  install_with_dnf
elif command -v yum >/dev/null 2>&1; then
  install_with_yum
elif command -v pacman >/dev/null 2>&1; then
  install_with_pacman
elif command -v zypper >/dev/null 2>&1; then
  install_with_zypper
elif command -v apk >/dev/null 2>&1; then
  install_with_apk
else
  echo "Unsupported package manager. Install dependencies manually: ${base_packages[*]} ${build_packages[*]}" >&2
  exit 1
fi
