#!/usr/bin/env bash
set -euo pipefail

if command -v sudo >/dev/null 2>&1 && [[ "${EUID:-$(id -u)}" -ne 0 ]]; then
  SUDO="sudo"
else
  SUDO=""
fi

tmp_dir="$(mktemp -d)"
trap 'rm -rf "$tmp_dir"' EXIT

curl -fsSL https://opencode.ai/install -o "$tmp_dir/opencode_install.sh"
$SUDO bash "$tmp_dir/opencode_install.sh"

if [[ ! -x "/usr/local/bin/opencode" ]]; then
  bin_path="$($SUDO find /root -type f -name opencode -perm -111 2>/dev/null | head -n 1 || true)"
  if [[ -n "$bin_path" ]]; then
    $SUDO install -m 0755 "$bin_path" "/usr/local/bin/opencode"
  fi
fi
