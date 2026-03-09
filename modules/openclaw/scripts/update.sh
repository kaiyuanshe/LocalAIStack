#!/usr/bin/env bash
set -euo pipefail

script_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# Reuse the installer helpers so update and install stay consistent.
export OPENCLAW_SKIP_MAIN=1
# shellcheck disable=SC1091
source "${script_dir}/install.sh"

install_openclaw_latest() {
  ensure_nodejs_for_npm

  if [[ "$(command -v npm)" == "${NVM_DIR}"/* ]]; then
    npm install -g openclaw@latest
  else
    $SUDO npm install -g openclaw@latest
  fi
}

main() {
  install_openclaw_latest

  command -v openclaw >/dev/null 2>&1 || {
    echo "OpenClaw CLI is not available in PATH after update." >&2
    exit 1
  }

  openclaw onboard --install-daemon
  bash "${script_dir}/configure_local.sh"

  echo "OpenClaw updated successfully."
}

main "$@"
