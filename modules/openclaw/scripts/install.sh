#!/usr/bin/env bash
set -euo pipefail

if command -v sudo >/dev/null 2>&1 && [[ "${EUID:-$(id -u)}" -ne 0 ]]; then
  SUDO="sudo"
else
  SUDO=""
fi

ensure_usr_local_bin() {
  $SUDO mkdir -p /usr/local/bin
}

has_openclaw() {
  command -v openclaw >/dev/null 2>&1
}

install_from_script() {
  local install_url="$1"
  local tmp_dir
  tmp_dir="$(mktemp -d)"
  trap 'rm -rf "$tmp_dir"' RETURN

  curl -fsSL "$install_url" -o "$tmp_dir/openclaw_install.sh"
  $SUDO bash "$tmp_dir/openclaw_install.sh"
}

install_from_npm() {
  if ! command -v npm >/dev/null 2>&1; then
    return 1
  fi

  $SUDO npm install -g @openclaw/cli || $SUDO npm install -g openclaw-cli
}

install_from_pip() {
  if command -v pipx >/dev/null 2>&1; then
    pipx install openclaw || pipx upgrade openclaw || true
    return 0
  fi

  if command -v python3 >/dev/null 2>&1; then
    python3 -m pip install --user --upgrade openclaw
    return 0
  fi

  return 1
}

promote_binary_to_usr_local() {
  if has_openclaw; then
    return 0
  fi

  local candidates=(
    "${HOME}/.local/bin/openclaw"
    "/root/.local/bin/openclaw"
  )

  for candidate in "${candidates[@]}"; do
    if [[ -x "$candidate" ]]; then
      ensure_usr_local_bin
      $SUDO install -m 0755 "$candidate" /usr/local/bin/openclaw
      return 0
    fi
  done

  return 1
}

main() {
  local install_url="${OPENCLAW_INSTALL_URL:-https://openclaw.ai/install.sh}"

  if has_openclaw; then
    echo "OpenClaw already installed: $(command -v openclaw)"
    return 0
  fi

  if install_from_script "$install_url" 2>/dev/null || true; then
    :
  fi

  if ! has_openclaw; then
    install_from_npm || true
  fi

  if ! has_openclaw; then
    install_from_pip || true
    promote_binary_to_usr_local || true
  fi

  has_openclaw || {
    echo "Failed to install OpenClaw. Set OPENCLAW_INSTALL_URL or install manually, then re-run verification." >&2
    exit 1
  }

  echo "OpenClaw installed at: $(command -v openclaw)"
}

main "$@"
