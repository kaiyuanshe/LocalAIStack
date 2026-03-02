#!/usr/bin/env bash
set -euo pipefail

if command -v sudo >/dev/null 2>&1 && [[ "${EUID:-$(id -u)}" -ne 0 ]]; then
  SUDO="sudo"
else
  SUDO=""
fi

tmp_dir="$(mktemp -d)"
trap 'rm -rf "$tmp_dir"' EXIT

curl -fsSL https://llmfit.axjns.dev/install.sh -o "$tmp_dir/llmfit_install.sh"
bash "$tmp_dir/llmfit_install.sh"

if ! command -v llmfit >/dev/null 2>&1; then
  for candidate in "${HOME}/.local/bin/llmfit" "/root/.local/bin/llmfit"; do
    if [[ -x "$candidate" ]]; then
      $SUDO install -m 0755 "$candidate" /usr/local/bin/llmfit
      break
    fi
  done
fi
