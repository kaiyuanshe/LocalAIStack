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

base_packages=(curl ca-certificates tar python3)
build_packages=()

if [[ "$mode" == "source" ]]; then
  build_packages=(git cmake make gcc g++)
fi

install_with_apt() {
  $SUDO apt-get update -y
  $SUDO apt-get install -y "${base_packages[@]}" "${build_packages[@]}"
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
