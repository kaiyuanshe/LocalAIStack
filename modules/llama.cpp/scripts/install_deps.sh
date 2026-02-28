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

if [[ "$mode" == "source" ]]; then
  build_packages=(git cmake make gcc g++)
fi

if [[ "$cuda_requested" -eq 1 ]]; then
  if [[ "$mode" == "source" ]]; then
    cuda_host_packages=(gcc-10 g++-10)
  fi
fi

install_with_apt() {
  if ! $SUDO apt-get update -y; then
    echo "apt-get update failed; retrying with only the default sources.list (ignoring sources.list.d entries)." >&2
    $SUDO apt-get update -y -o Dir::Etc::sourceparts="-"
  fi
  $SUDO apt-get install -y "${base_packages[@]}" "${build_packages[@]}" "${cuda_host_packages[@]}"
}

install_with_dnf() {
  $SUDO dnf install -y "${base_packages[@]}" "${build_packages[@]/g++/gcc-c++}"
}

install_with_yum() {
  $SUDO yum install -y "${base_packages[@]}" "${build_packages[@]/g++/gcc-c++}"
}

install_with_pacman() {
  $SUDO pacman -Sy --noconfirm "${base_packages[@]}" "${build_packages[@]}"
}

install_with_zypper() {
  $SUDO zypper --non-interactive install "${base_packages[@]}" "${build_packages[@]/g++/gcc-c++}"
}

install_with_apk() {
  $SUDO apk add --no-cache "${base_packages[@]}" "${build_packages[@]}"
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
